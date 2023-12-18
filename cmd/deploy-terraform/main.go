package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/opendevstack/ods-pipeline/pkg/logging"
	"github.com/opendevstack/ods-pipeline/pkg/pipelinectxt"
	"k8s.io/client-go/kubernetes"
)

const (
	terraformBin                = "terraform"
	kubernetesServiceaccountDir = "/var/run/secrets/kubernetes.io/serviceaccount"
)

type options struct {
	// Location of checkout directory.
	checkoutDir string
	// Location of terraform files directory.
	terraformDir string
	// Terraform Kubernetes Backend secret_suffix will be incorporated here..
	targetEnvironment string
	// Whether to derive env variables from the k8s secret
	envFromSecret bool
	// Whether to apply or plan only without changing existing resources.
	planOnly bool
	// terraform apply extra args
	applyExtraArgs string
	// terraform plan extra args
	planExtraArgs string
	// Whether to enable debug mode.
	debug bool
	// Whether to enable verbose mode.
	verbose bool
}

type terraformConfig struct {
	// subrepo is nil if this is about terraform config in this repo
	subrepo          fs.DirEntry
	subrepoArtifacts []string
	// directory where terraform config (*.tf) files are located at.
	terraformDir string
	// artifact name
	artifactName string
}

func (t terraformConfig) String() string {
	result := "terraformConfig{"
	if t.subrepo != nil {
		result += fmt.Sprintf("subrepo: %v, ", t.subrepo)
	}
	if t.terraformDir != "" {
		result += fmt.Sprintf("terraformDir: %s, ", t.terraformDir)
	}
	if t.artifactName != "" {
		result += fmt.Sprintf("artifactName: %s, ", t.artifactName)
	}
	result += "}"
	return result
}

type deployTerraform struct {
	logger logging.LeveledLoggerInterface
	// Name of terraform binary.
	terraformBin        string
	opts                *options
	ctxt                *pipelinectxt.ODSContext
	clientset           *kubernetes.Clientset
	secretName          string
	secretEnvVars       map[string]string
	pluginCacheDir      string
	subrepos            []fs.DirEntry
	deploymentArtifacts []string
	tfConfigs           []terraformConfig
	outWriter           io.Writer
	errWriter           io.Writer
}

var defaultOptions = options{
	checkoutDir:       ".",
	terraformDir:      "./terraform",
	targetEnvironment: "dev",
	envFromSecret:     true,
	planOnly:          false,
	applyExtraArgs:    "",
	planExtraArgs:     "",
	debug:             (os.Getenv("DEBUG") == "true"),
	verbose:           false,
}

func deployTerraformFromOptions(opts *options, out, err io.Writer) *deployTerraform {
	var logger logging.LeveledLoggerInterface
	if opts.debug {
		logger = &logging.LeveledLogger{Level: logging.LevelDebug}
	} else {
		logger = &logging.LeveledLogger{Level: logging.LevelInfo}
	}

	return &deployTerraform{
		terraformBin: terraformBin,
		logger:       logger,
		opts:         opts,
		outWriter:    out,
		errWriter:    err,
	}
}

func main() {
	opts := options{}
	flag.StringVar(&opts.checkoutDir, "checkout-dir", defaultOptions.checkoutDir, "Checkout dir")
	flag.StringVar(&opts.terraformDir, "terraform-dir", defaultOptions.terraformDir, "Terraform files directory")
	flag.StringVar(&opts.targetEnvironment, "target-environment", defaultOptions.targetEnvironment, "Identified target environment for terraform resources to apply to. Also used in the name of the Terraform state file (tfstate-{terraform-workspace}-{target-environment})")
	flag.BoolVar(&opts.envFromSecret, "env-from-secret", defaultOptions.envFromSecret, "Whether to derive env variables from the k8s secret terraform-var-`target-environment`")
	flag.BoolVar(&opts.planOnly, "plan-only", defaultOptions.planOnly, "Whether to perform only a terraform plan")
	flag.StringVar(&opts.applyExtraArgs, "apply-extra-args", defaultOptions.applyExtraArgs, "Extra arguments to pass to `terraform apply`")
	flag.StringVar(&opts.planExtraArgs, "plan-extra-args", defaultOptions.planExtraArgs, "Extra arguments to pass to `terraform plan`")
	flag.BoolVar(&opts.debug, "debug", defaultOptions.debug, "debug mode enables debug loggers and debug parameter passed into executed commands if available.")
	flag.BoolVar(&opts.verbose, "verbose", defaultOptions.verbose, "verbose mode. debug implies verbose.")
	flag.Parse()

	dt := deployTerraformFromOptions(&opts, os.Stdout, os.Stderr)
	err := (dt).runSteps(
		setupContext(),
		setupEnvFromSecret(),
		renderBackend(),
		detectSubrepos(),
		detectDeploymentArtifacts(),
		locateTerraformConfigs(),
		initTerraform(),
		planTerraform(),
		applyTerraform(),
	)
	if err != nil {
		dt.logger.Errorf(err.Error())
		os.Exit(1)
	}
}
