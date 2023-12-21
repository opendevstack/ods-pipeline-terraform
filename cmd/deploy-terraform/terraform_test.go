package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opendevstack/ods-pipeline/pkg/pipelinectxt"
)

func TestArgEnvs(t *testing.T) {
	tests := map[string]struct {
		opts           options
		ctxtNamespace  string
		pluginCacheDir string
		wantErr        bool
		wantArgs       []string
		wantEnv        map[string]string
		wantSensitive  []string
	}{
		"init args/env test": {
			opts: options{
				checkoutDir:       "../../test/testdata/workspaces/terraform-sample",
				terraformDir:      "../../test/testdata/workspaces/terraform-sample",
				targetEnvironment: "dev",
				planOnly:          true,
				applyExtraArgs:    "",
				planExtraArgs:     "",
				debug:             false,
			},
			ctxtNamespace:  "namespace",
			pluginCacheDir: "../../test/pluginCache",
			wantErr:        false,
			wantArgs:       []string{"init", "-input=false", "-no-color"},
			wantEnv: map[string]string{
				"KUBE_NAMESPACE": "namespace",
			},
			wantSensitive: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var stdoutMulti, stderrMulti bytes.Buffer
			bbStdoutWriter := io.MultiWriter(os.Stdout, &stdoutMulti)
			bbStderrWriter := io.MultiWriter(os.Stderr, &stderrMulti)
			d := deployTerraformFromOptions(&tc.opts, bbStdoutWriter, bbStderrWriter)
			testPluginCachedDir, err := filepath.Abs(tc.pluginCacheDir)
			if err != nil {
				t.Fatalf("want no err, got %s", err)
			}
			d.pluginCacheDir = testPluginCachedDir
			d.ctxt = &pipelinectxt.ODSContext{
				Namespace: tc.ctxtNamespace,
			}

			args, env, sensitive, err := d.assembleInitWithK8sBackendArgsEnv()
			if tc.wantErr && err == nil {
				t.Fatal("want err, got none")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("want no err, got %s", err)
			}
			if diff := cmp.Diff(tc.wantArgs, args); diff != "" {
				t.Fatalf("args mismatch (-want +got):\n%s", diff)
			}
			tc.wantEnv["TF_PLUGIN_CACHE_DIR"] = testPluginCachedDir
			if diff := cmp.Diff(tc.wantEnv, env); diff != "" {
				t.Fatalf("env mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantSensitive, sensitive); diff != "" {
				t.Fatalf("sensitive mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
