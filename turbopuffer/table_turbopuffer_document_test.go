package turbopuffer

// Unit checks for document id rendering. turbopuffer document ids are
// string | int64; these prove integer ids never render in scientific
// notation or lose precision, whatever numeric shape the decoder yields.

import (
	"encoding/json"
	"testing"

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
