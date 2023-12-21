package output

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMaskOutput(t *testing.T) {
	tests := map[string]struct {
		original  string
		sensitive []string
		wanted    string
	}{
		"masking sensitive info": {
			original: "sensitive=password123 and another TEST_API=s3cr3t",
			sensitive: []string{
				"password123", "s3cr3t",
			},
			wanted: "sensitive=*** and another TEST_API=***",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Creating a buffer as an example output destination
			var buf bytes.Buffer
			maskWriter := NewMaskWriter(&buf, tc.sensitive)
			maskWriter.Write([]byte(tc.original))
			got := buf.String()
			if diff := cmp.Diff(tc.wanted, got); diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}

}
