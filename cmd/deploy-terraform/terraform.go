package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/shlex"
	"github.com/opendevstack/ods-pipeline-terraform/internal/command"
	"github.com/opendevstack/ods-pipeline-terraform/internal/kubernetes"
	"github.com/opendevstack/ods-pipeline-terraform/internal/output"
)

const (
	// exit code returned from terraform plan
	planSuccessNoChangesExitCode   = 0 // (diff empty)
	planErrorExitCode              = 1
	planSuccessWithChangesExitCode = 2 // (diff not empty)
)

func (d *deployTerraform) terraformCmd(args []string, env map[string]string, dir string, outWriter, errWriter io.Writer) error {

	return command.RunInDir(
		d.terraformBin, args, env, dir, outWriter, errWriter,
	)
}

// terraform plan runs the diff and returns whether the plan is in sync.
// An error is returned when the plan cannot be started or encounters failures
func (d *deployTerraform) terraformPlanInSync(args []string, env map[string]string, dir string, outWriter, errWriter io.Writer) (bool, error) {
	return command.RunWithSpecialFailureCode(
		d.terraformBin, args, env, dir, outWriter, errWriter, planSuccessWithChangesExitCode,
	)
}

func (d *deployTerraform) assembleInitWithK8sBackendArgsEnv() (args []string, env map[string]string, sensitive []string, err error) {
	args = []string{
		"init",
	}
	commonArgs := d.commonTerraformArgs()
	env = d.commonTerraformEnv()
	env["TF_PLUGIN_CACHE_DIR"] = d.pluginCacheDir
	sensitive = []string{}
	for k, v := range d.secretEnvVars {
		env[k] = v
		sensitive = append(sensitive, v)
	}
	return append(args, commonArgs...), env, sensitive, nil
}

// assemblePlanArgs creates a slice of arguments for "terraform plan".
func (d *deployTerraform) assemblePlanArgsEnv() (args []string, env map[string]string, sensitive []string, err error) {
	empty := []string{}
	emptyEnv := make(map[string]string)
	args = []string{
		"plan",
		"-detailed-exitcode",
	}
	planExtraArgs, err := shlex.Split(d.opts.planExtraArgs)
	if err != nil {
		return empty, emptyEnv, empty, fmt.Errorf("parse plan-extra-args (%s): %s", d.opts.planExtraArgs, err)
	}
	args = append(args, planExtraArgs...)
	commonArgs := d.commonTerraformPlanApplyArgs()
	env = d.commonTerraformPlanApplyEnv()
	sensitive = []string{}
	for k, v := range d.secretEnvVars {
		env[k] = v
		sensitive = append(sensitive, v)
	}
	return append(args, commonArgs...), env, sensitive, nil
}

// assembleApplyArgs creates a slice of arguments for "terraform apply".
func (d *deployTerraform) assembleApplyArgsEnv() (args []string, env map[string]string, sensitive []string, err error) {
	empty := []string{}
	emptyEnv := make(map[string]string)
	args = []string{
		"apply",
		"-auto-approve", // should save plan instead and use it
	}
	applyExtraArgs, err := shlex.Split(d.opts.applyExtraArgs)
	if err != nil {
		return empty, emptyEnv, empty, fmt.Errorf("parse apply-extra-args (%s): %s", d.opts.applyExtraArgs, err)
	}
	args = append(args, applyExtraArgs...)
	commonArgs := d.commonTerraformPlanApplyArgs()
	env = d.commonTerraformPlanApplyEnv()
	sensitive = []string{}
	for k, v := range d.secretEnvVars {
		env[k] = v
		sensitive = append(sensitive, v)
	}
	return append(args, commonArgs...), env, sensitive, nil
}

// commonTerraformPlanApplyArgs returns arguments common to "terraform upgrade" and "terraform diff upgrade".
func (d *deployTerraform) commonTerraformPlanApplyArgs() []string {
	args := d.commonTerraformArgs()
	apArgs := []string{
		"-compact-warnings",
	}
	args = append(args, apArgs...)
	return args
}

func (d *deployTerraform) commonTerraformPlanApplyEnv() map[string]string {
	args := d.commonTerraformEnv()
	return args
}

// commonTerraformArgs returns arguments common to any Terraform command.
func (d *deployTerraform) commonTerraformArgs() []string {
	args := []string{
		"-input=false",
		"-no-color",
	}
	// https://support.hashicorp.com/hc/en-us/articles/360001113727-Enabling-trace-level-logs-in-Terraform-CLI-Cloud-or-Enterprise
	// todo: is there some kind of debug flag?
	// if d.opts.debug {
	// 	args = append([]string{"-debug"}, args...)
	// }
	return args
}

func (d *deployTerraform) commonTerraformEnv() map[string]string {
	envs := make(map[string]string)
	envs["KUBE_NAMESPACE"] = d.ctxt.Namespace
	return envs
}

func (d *deployTerraform) terraformSecrets() (map[string]string, error) {
	d.secretName = fmt.Sprintf("terraform-envs-%s", d.opts.targetEnvironment)
	envs, err := kubernetes.GetSecrets(d.clientset, d.ctxt.Namespace, d.secretName)
	if err != nil {
		return nil, fmt.Errorf("getting secret '%s' in namespace %s failed: %w)", d.secretName, d.ctxt.Namespace, err)
	}
	return envs, nil
}

func getCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("E:%s", err)
	} else {
		return cwd
	}
}

func dirInfo(dir string) string {
	return strings.Join([]string{
		fmt.Sprintf("pwd=%s", getCwd()),
		fmt.Sprintf("dir=%s", dir),
	}, ", ") + ":"
}

func formatEnv(env map[string]string) string {
	envlist := []string{}
	for k, v := range env {
		envlist = append(envlist, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(envlist, " ")
}

func printlnTerraformCmd(args []string, env map[string]string, sensitive []string, dir string, outWriter io.Writer) {
	maskWriter := output.NewMaskWriter(outWriter, sensitive)

	fmt.Fprintln(maskWriter, strings.Join([]string{
		dirInfo(dir),
		formatEnv(env),
		terraformBin,
		strings.Join(args, " "),
	}, " "))
}
