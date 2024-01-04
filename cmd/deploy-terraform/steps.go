package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/opendevstack/ods-pipeline-terraform/internal/command"
	"github.com/opendevstack/ods-pipeline/pkg/pipelinectxt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	tokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

type TerraformStep func(d *deployTerraform) (*deployTerraform, error)

func (d *deployTerraform) runSteps(steps ...TerraformStep) error {
	var skip *skipRemainingSteps
	var err error
	for _, step := range steps {
		d, err = step(d)
		if err != nil {
			if errors.As(err, &skip) {
				d.logger.Infof(err.Error())
				return nil
			}
			return err
		}
	}
	return nil
}
func (d *deployTerraform) isVerbose() bool {
	return d.opts.debug || d.opts.verbose
}

func setupContext() TerraformStep {
	return func(d *deployTerraform) (*deployTerraform, error) {
		ctxt := &pipelinectxt.ODSContext{}
		err := ctxt.ReadCache(d.opts.checkoutDir)
		if err != nil {
			return d, fmt.Errorf("read cache: %w", err)
		}
		d.ctxt = ctxt

		clientset, err := newInClusterClientset()
		if err != nil {
			return d, fmt.Errorf("create Kubernetes clientset: %w", err)
		}
		d.clientset = clientset

		if d.isVerbose() {
			err = command.Run("sh", []string{
				"-c",
				"env | sort",
			}, make(map[string]string), d.outWriter, d.errWriter)
			if err != nil {
				d.logger.Infof("env command failed: %w", err)
			}
		}
		err = os.MkdirAll(pipelinectxt.DeploymentsPath, 0755)
		if err != nil {
			return d, fmt.Errorf("create artifact path: %w", err)
		}
		cacheDir := filepath.Join(pipelinectxt.BaseDir, "deps", "terraform")
		d.pluginCacheDir, err = filepath.Abs(cacheDir)
		if err != nil {
			return d, fmt.Errorf("create TF_PLUGIN_CACHE_DIR value failed for %s: %w", cacheDir, err)
		}
		err = os.MkdirAll(d.pluginCacheDir, os.ModePerm)
		if err != nil {
			return d, fmt.Errorf("TF_PLUGIN_CACHE_DIR could not be created at %s: %w", d.pluginCacheDir, err)
		}
		return d, nil
	}
}

func renderBackend() TerraformStep {
	return func(d *deployTerraform) (*deployTerraform, error) {
		d.logger.Infof("rendering backend template %s...", d.opts.terraformDir)
		err := d.renderBackend()
		if err != nil {
			return d, fmt.Errorf("render backend template failed: %w", err)
		}
		return d, nil
	}
}

func setupEnvFromSecret() TerraformStep {
	return func(d *deployTerraform) (*deployTerraform, error) {
		if d.opts.envFromSecret {
			d.logger.Infof("deriving env variables from kubernetes secret")
			envVars, err := d.terraformSecrets()
			if err != nil {
				return d, err
			}
			d.secretEnvVars = envVars
			d.logger.Infof("Secret env variables: [%s]", strings.Join(getKeys(envVars), ","))
		} else {
			d.logger.Infof("env-from-secret is %s: skipping deriving env variables from kubernetes secret")
		}
		return d, nil
	}
}

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func initTerraform() TerraformStep {
	return func(d *deployTerraform) (*deployTerraform, error) {
		for _, tfConfig := range d.tfConfigs {
			dir := tfConfig.terraformDir
			d.logger.Infof("terraform init %s...", dir)

			initArgs, initEnv, sensitive, err := d.assembleInitWithK8sBackendArgsEnv()
			if err != nil {
				return d, fmt.Errorf("assemble terraform init args/env: %w", err)
			}
			printlnTerraformCmd(initArgs, initEnv, sensitive, dir, d.outWriter)
			err = d.terraformCmd(initArgs, initEnv, dir, d.outWriter, d.errWriter)
			if err != nil {
				return d, fmt.Errorf("terraform init: %w", err)
			}
		}
		return d, nil
	}
}

func detectSubrepos() TerraformStep {
	return func(d *deployTerraform) (*deployTerraform, error) {
		subrepos, err := pipelinectxt.DetectSubrepos()
		if err != nil {
			return d, fmt.Errorf("detect subrepos: %w", err)
		}
		d.subrepos = subrepos
		d.logger.Infof("Detecting sub repositories %s", d.subrepos)
		return d, nil
	}
}

func detectDeploymentArtifacts() TerraformStep {
	return func(d *deployTerraform) (*deployTerraform, error) {
		deploymentArtifacts, err := pipelinectxt.ReadArtifactFilesIncludingSubrepos(pipelinectxt.DeploymentsPath, d.subrepos)
		if err != nil {
			return d, fmt.Errorf("collect deployment artifacts: %w", err)
		}
		d.deploymentArtifacts = deploymentArtifacts
		return d, nil
	}
}

func (d *deployTerraform) isTerraformDir(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		d.logger.Debugf("No terraform directory found at %s: %w", dir, err)
		return false
	}
	// perhaps additional checks make sense to see whether dir is a terraform dir
	return true
}

func locateTerraformConfigs() TerraformStep {
	return func(d *deployTerraform) (*deployTerraform, error) {
		tfConfigs := []terraformConfig{}
		d.logger.Infof("Looking for terraform configs in directory '%s' ...", d.opts.terraformDir)
		if d.isTerraformDir(d.opts.terraformDir) {
			tfConfig := terraformConfig{
				terraformDir: d.opts.terraformDir,
				artifactName: artifactFilename("plan", d.opts.terraformDir, d.opts.targetEnvironment),
			}
			tfConfigs = append(tfConfigs, tfConfig)
			d.logger.Infof("Located %s ", tfConfig)
		}

		// Find terraform configs in subrepos
		for _, r := range d.subrepos {
			subrepo := filepath.Join(pipelinectxt.SubreposPath, r.Name())
			subTerraformDir := filepath.Join(subrepo, d.opts.terraformDir)
			if d.isTerraformDir(subTerraformDir) {
				d.logger.Infof("No terraform config found at %s", subTerraformDir)
				continue
			}
			d.logger.Infof("Located terraform config at %s directory", subTerraformDir)
			deploymentArtifacts, err := pipelinectxt.ReadArtifactFilesIncludingSubrepos(pipelinectxt.DeploymentsPath, []fs.DirEntry{r})
			if err != nil {
				return d, fmt.Errorf("collect deployment artifacts: %w", err)
			}
			tfConfig := terraformConfig{
				terraformDir:     subTerraformDir,
				artifactName:     artifactFilename("plan", d.opts.terraformDir, d.opts.targetEnvironment),
				subrepo:          r,
				subrepoArtifacts: deploymentArtifacts,
			}
			tfConfigs = append(tfConfigs, tfConfig)
			d.logger.Infof("Located subrepo  %s ", tfConfig)

		}
		d.tfConfigs = tfConfigs
		return d, nil
	}
}

func planTerraform() TerraformStep {
	return func(d *deployTerraform) (*deployTerraform, error) {
		for _, tfConfig := range d.tfConfigs {
			dir := tfConfig.terraformDir
			d.logger.Infof("terraform plan %s...", dir)
			planArgs, planEnv, sensitive, err := d.assemblePlanArgsEnv()
			if err != nil {
				return d, fmt.Errorf("assemble terraform plan args: %w", err)
			}
			printlnTerraformCmd(planArgs, planEnv, sensitive, dir, d.outWriter)
			var planStdoutBuf bytes.Buffer
			planStdoutWriter := io.MultiWriter(d.outWriter, &planStdoutBuf)
			inSync, err := d.terraformPlanInSync(planArgs, planEnv, dir, planStdoutWriter, d.errWriter)
			if err != nil {
				return d, fmt.Errorf("terraform plan: %w", err)
			}
			err = d.writeDeploymentArtifact(planStdoutBuf.Bytes(), tfConfig, d.opts.targetEnvironment)
			if err != nil {
				return d, fmt.Errorf("write plan artifact: %w", err)
			}

			if d.opts.planOnly {
				return d, &skipRemainingSteps{"Only planning was requested, skipping terraform apply."}
			}
			if inSync {
				return d, &skipRemainingSteps{"No changes detected, skipping terraform apply."}
			}
		}
		return d, nil
	}
}

func applyTerraform() TerraformStep {
	return func(d *deployTerraform) (*deployTerraform, error) {
		for _, tfConfig := range d.tfConfigs {
			dir := tfConfig.terraformDir
			d.logger.Infof("terraform plan to %s...", dir)
			applyArgs, applyEnv, sensitive, err := d.assembleApplyArgsEnv()
			if err != nil {
				return d, fmt.Errorf("assemble terraform apply args: %w", err)
			}
			printlnTerraformCmd(applyArgs, applyEnv, sensitive, dir, d.outWriter)
			err = d.terraformCmd(applyArgs, applyEnv, dir, d.outWriter, d.errWriter)
			if err != nil {
				return d, fmt.Errorf("terraform apply: %w", err)
			}
		}
		return d, nil
	}
}

func (d *deployTerraform) writeDeploymentArtifact(content []byte, tfConfig terraformConfig, targetEnv string) error {
	var f string
	if tfConfig.subrepo != nil {
		f = fmt.Sprintf("%s-%s.txt", tfConfig.subrepo.Name(), tfConfig.artifactName)
	} else {
		f = tfConfig.artifactName + ".txt"
	}
	file := filepath.Join(pipelinectxt.DeploymentsPath, f)
	err := os.WriteFile(file, content, 0644)
	if err == nil {
		d.logger.Infof("wrote artifact %s", f)
	}
	return err
}

func artifactFilename(filename, terraformDir, targetEnv string) string {
	trimmedTerraformDir := strings.TrimPrefix(terraformDir, "./")
	if trimmedTerraformDir != "terraform" {
		filename = fmt.Sprintf("%s-%s", strings.Replace(trimmedTerraformDir, "/", "-", -1), filename)
	}
	return fmt.Sprintf("%s-%s", filename, targetEnv)
}

func newInClusterClientset() (*kubernetes.Clientset, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	return kubernetes.NewForConfig(config)
}
