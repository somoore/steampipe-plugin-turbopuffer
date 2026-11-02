package turbopuffer

// Regression tests for the security-relevant helpers: region validation (the
// value is substituted into the API base URL, so it must not be able to move
// the host the bearer key is sent to), recall bounds (billable requests), and
// the query error policy.

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	tpuf "github.com/turbopuffer/turbopuffer-go/v2"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

func TestValidRegion(t *testing.T) {
	valid := []string{"gcp-us-central1", "aws-us-east-1", "a", "a1", "us-east-1"}
	for _, r := range valid {
		if !validRegion(r) {
			t.Errorf("validRegion(%q) = false, want true", r)
		}
	}

	// Every one of these would rewrite or escape the
	// https://REGION.turbopuffer.com/ template.
	invalid := []string{
		"", " ", "evil.example/", "evil.example", "https://evil.example",
		"a@b", "a b", "a?b", "a#b", "../x", "x/y", "gcp-us-central1/x",
		"-leading", "trailing-", "a_b", "GCP-US-CENTRAL1", ".", "..",
		strings.Repeat("a", 64),
	}
	for _, r := range invalid {
		if validRegion(r) {
			t.Errorf("validRegion(%q) = true, want false", r)
		}
	}
}

func TestConfiguredRegionsRejectsHostEscape(t *testing.T) {
	got := configuredRegions(turbopufferConfig{Regions: []string{"evil.example/"}})
	if len(got) != 1 || got[0] != defaultRegion {
		t.Fatalf("configuredRegions([evil.example/]) = %v, want [%s]", got, defaultRegion)
	}

	got = configuredRegions(turbopufferConfig{Regions: []string{"evil.example/", "a@b", ""}})
	if len(got) != 1 || got[0] != defaultRegion {
		t.Fatalf("all-invalid config = %v, want [%s]", got, defaultRegion)
	}

	// Valid values keep the existing trim/dedupe/order behaviour, and invalid
	// neighbours do not poison them.
	got = configuredRegions(turbopufferConfig{Regions: []string{
		" gcp-us-central1 ", "evil.example/", "aws-us-east-1", "gcp-us-central1",
	}})
	if len(got) != 2 || got[0] != "gcp-us-central1" || got[1] != "aws-us-east-1" {
		t.Fatalf("mixed config = %v, want [gcp-us-central1 aws-us-east-1]", got)
	}

	if got := configuredRegions(turbopufferConfig{}); len(got) != 1 || got[0] != defaultRegion {
		t.Fatalf("empty config = %v, want [%s]", got, defaultRegion)
	}
}

func TestRejectedRegions(t *testing.T) {
	cfg := turbopufferConfig{Regions: []string{"gcp-us-central1", " evil.example/ ", "a@b", ""}}
	got := rejectedRegions(cfg)
	if len(got) != 2 || got[0] != "evil.example/" || got[1] != "a@b" {
		t.Fatalf("rejectedRegions = %v, want [evil.example/ a@b]", got)
	}
}

func TestGetClientRejectsInvalidRegion(t *testing.T) {
	// A zero QueryData has a nil cache: validation must run before any cache
	// or credential access, so this must return an error, not panic.
	_, err := getClient(context.Background(), &plugin.QueryData{}, "evil.example/")
	if err == nil || !strings.Contains(err.Error(), "invalid region") {
		t.Fatalf("getClient(invalid region) err = %v, want an invalid region error", err)
	}
}

func TestRecallBound(t *testing.T) {
	cases := []struct {
		v    int64
		want bool
	}{
		{0, false}, {-1, false}, {1, true}, {10, true}, {1000, true},
		{1001, false}, {math.MaxInt64, false},
	}
	for _, c := range cases {
		if _, err := recallBound("queries", c.v, maxRecallQueries); (err == nil) != c.want {
			t.Errorf("recallBound(queries, %d) err = %v, want valid=%v", c.v, err, c.want)
		}
	}
}

func TestRecallRejectsOutOfRangeQualBeforeAnyRequest(t *testing.T) {
	// d has no connection cache and no credentials: if validation happened
	// after getClient this would panic or fail on config rather than on the
	// bound. The error must name the range.
	cases := []struct {
		col   string
		value int64
	}{
		{"queries", 0}, {"queries", -1}, {"queries", 1001}, {"queries", math.MaxInt64},
		{"top_k", 0}, {"top_k", -1}, {"top_k", 1001}, {"top_k", math.MaxInt64},
	}
	for _, c := range cases {
		d := &plugin.QueryData{EqualsQuals: plugin.KeyColumnEqualsQualMap{
			"namespace": &proto.QualValue{Value: &proto.QualValue_StringValue{StringValue: "ns"}},
			c.col:       &proto.QualValue{Value: &proto.QualValue_Int64Value{Int64Value: c.value}},
		}}
		_, err := listNamespaceRecall(context.Background(), d, nil)
		if err == nil || !strings.Contains(err.Error(), "between 1 and") {
			t.Errorf("%s=%d: err = %v, want a bounds error", c.col, c.value, err)
		}
	}
}

func TestErrorPolicy(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		code   int
		ignore bool
		retry  bool
	}{
		{400, false, false},
		{404, true, false},
		{429, false, true},
		{500, false, true},
		{503, false, true},
	}
	for _, c := range cases {
		err := &tpuf.Error{StatusCode: c.code}
		if got := isNotFoundError(ctx, nil, nil, err); got != c.ignore {
			t.Errorf("isNotFoundError(%d) = %v, want %v", c.code, got, c.ignore)
		}
		if got := shouldRetryError(ctx, nil, nil, err); got != c.retry {
			t.Errorf("shouldRetryError(%d) = %v, want %v", c.code, got, c.retry)
		}
	}
	plain := errors.New("boom")
	if isNotFoundError(ctx, nil, nil, plain) {
		t.Error("isNotFoundError(non-API error) = true, want false")
	}
	if shouldRetryError(ctx, nil, nil, plain) {
		t.Error("shouldRetryError(non-API error) = true, want false (no blind retries)")
	}
}

func TestHydratesRejectUnexpectedItemType(t *testing.T) {
	// A wrong item type must be an error, never a panic that kills the plugin.
	ctx := context.Background()
	h := &plugin.HydrateData{Item: "not-a-namespace-row"}
	if _, err := getNamespaceMetadata(ctx, &plugin.QueryData{}, h); err == nil {
		t.Error("getNamespaceMetadata(wrong item type) returned no error")
	}
	if _, err := listNamespaceAttributes(ctx, &plugin.QueryData{}, h); err == nil {
		t.Error("listNamespaceAttributes(wrong item type) returned no error")
	}
}
