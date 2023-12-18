package main

import (
	"testing"
)

func TestArtifactFilename(t *testing.T) {
	tests := map[string]struct {
		filename     string
		terraformDir string
		targetEnv    string
		want         string
	}{
		"default terraform dir": {
			filename:     "plan",
			terraformDir: "./terraform",
			targetEnv:    "foo-dev",
			want:         "plan-foo-dev",
		},
		"default terraform dir without prefix": {
			filename:     "plan",
			terraformDir: "terraform",
			targetEnv:    "dev",
			want:         "plan-dev",
		},
		"other terraform dir": {
			filename:     "plan",
			terraformDir: "./foo-terraform",
			targetEnv:    "qa",
			want:         "foo-terraform-plan-qa",
		},
		"other terraform dir without prefix": {
			filename:     "plan",
			terraformDir: "bar-terraform",
			targetEnv:    "foo-qa",
			want:         "bar-terraform-plan-foo-qa",
		},
		"nested terraform dir": {
			filename:     "plan",
			terraformDir: "./some/path/terraform",
			targetEnv:    "prod",
			want:         "some-path-terraform-plan-prod",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := artifactFilename(tc.filename, tc.terraformDir, tc.targetEnv)
			if got != tc.want {
				t.Fatalf("want: %s, got: %s", tc.want, got)
			}
		})
	}
}
