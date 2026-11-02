package turbopuffer

// Unit checks for document id rendering and vector redaction. turbopuffer
// document ids are string | int64; these prove integer ids never render in
// scientific notation or lose precision, whatever numeric shape the decoder
// yields. The vector tests lock in the "vectors are never returned" guarantee
// (dense [N]f16/f32 and sparse {}f16).

import (
	"encoding/json"
	"testing"

	tpuf "github.com/turbopuffer/turbopuffer-go/v2"
	"github.com/turbopuffer/turbopuffer-go/v2/packages/respjson"
)

func TestDocumentID(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want string
	}{
		{"string id", "canary-001", "canary-001"},
		{"uuid string", "3c9f3f9e-3f62-4a1a-9d6b-2f1f0e6f6a01", "3c9f3f9e-3f62-4a1a-9d6b-2f1f0e6f6a01"},
		// The review's exact concern: fmt.Sprint(float64(1e6)) == "1e+06".
		{"float64 million", float64(1000000), "1000000"},
		{"float64 small", float64(42), "42"},
		// SDK decodes untyped JSON numbers as respjson.Number (raw text),
		// so even ids beyond 2^53 survive verbatim.
		{"respjson.Number beyond 2^53", respjson.Number("9007199254740993"), "9007199254740993"},
		{"json.Number", json.Number("123456789"), "123456789"},
		{"int64", int64(9007199254740993), "9007199254740993"},
	}
	for _, c := range cases {
		if got := documentID(c.in); got != c.want {
			t.Errorf("%s: documentID(%#v) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// TestIsVectorType locks in the "vectors are never returned" guarantee. Sparse
// vectors are {}f16, which does not start with "[", so the old predicate let
// them through.
func TestIsVectorType(t *testing.T) {
	cases := map[string]bool{
		// Dense vectors.
		"[1536]f32": true, "[768]f16": true, "[]f32": true, "[][8]f32": true,
		// Sparse vectors.
		"{}f16": true, "{}f32": true,
		// Not a current API type, but strip future float widths.
		"[8]f64": true, "[8]bf16": true,
		// Scalars are not vectors.
		"string": false, "int": false, "uint": false, "float": false,
		"bool": false, "uuid": false, "datetime": false,
		// Scalar arrays are legitimate (filterable) values, not vectors.
		"[]string": false, "[]int": false, "[]uint": false, "[]float": false,
		"[]bool": false, "[]uuid": false, "[]datetime": false,
		// Malformed/unknown types are left alone.
		"": false, "[nope]": false,
	}
	for typ, want := range cases {
		if got := isVectorType(typ); got != want {
			t.Errorf("isVectorType(%q) = %v, want %v", typ, got, want)
		}
	}
}

func TestDocumentAttributesRedactsVectors(t *testing.T) {
	row := tpuf.Row{
		"id":        "doc-1",
		"vector":    []float64{0.1},
		"embedding": []float64{0.2},
		"sparse":    map[string]interface{}{"a": 1.0},
		"acl":       "tenant-a",
		"tags":      []interface{}{"one", "two"},
	}
	attrs := documentAttributes(row, map[string]bool{"embedding": true, "sparse": true})

	for _, key := range []string{"id", "vector", "embedding", "sparse"} {
		if _, ok := attrs[key]; ok {
			t.Errorf("documentAttributes kept %q, want it stripped", key)
		}
	}
	if attrs["acl"] != "tenant-a" {
		t.Errorf("documentAttributes dropped acl: %v", attrs)
	}
	if len(attrs) != 2 {
		t.Errorf("documentAttributes = %v, want only acl and tags", attrs)
	}
}

func TestEffectiveTopK(t *testing.T) {
	zero, negative, five, huge := int64(0), int64(-1), int64(5), int64(500)
	cases := []struct {
		limit *int64
		want  int64
	}{
		{nil, sampleCap},
		{&zero, sampleCap},     // LIMIT 0 must not become top_k=0
		{&negative, sampleCap}, // nor a negative top_k
		{&five, 5},
		{&huge, sampleCap},
	}
	for _, c := range cases {
		if got := effectiveTopK(c.limit); got != c.want {
			t.Errorf("effectiveTopK(%v) = %d, want %d", c.limit, got, c.want)
		}
	}
}
