package e2e

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	ott "github.com/opendevstack/ods-pipeline/pkg/odstasktest"
	ttr "github.com/opendevstack/ods-pipeline/pkg/tektontaskrun"
)

var (
	namespaceConfig *ttr.NamespaceConfig
	rootPath        = "../.."
	taskName        = "ods-pipeline-terraform-deploy"
)

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	cc, err := ttr.StartKinDCluster(
		ttr.LoadImage(ttr.ImageBuildConfig{
			Dockerfile: "build/images/Dockerfile.terraform",
			ContextDir: rootPath,
		}),
	)
	if err != nil {
		log.Fatal("Could not start KinD cluster: ", err)
	}
	nc, cleanup, err := ttr.SetupTempNamespace(
		cc,
		ott.InstallODSPipeline(nil),
		ttr.InstallTaskFromPath(
			filepath.Join(rootPath, "build/tasks/deploy.yaml"),
			nil,
		),
	)
	if err != nil {
		log.Fatal("Could not setup temporary namespace: ", err)
	}
	defer cleanup()
	namespaceConfig = nc
	return m.Run()
}

func runTask(opts ...ttr.TaskRunOpt) error {
	return ttr.RunTask(append([]ttr.TaskRunOpt{
		ttr.InNamespace(namespaceConfig.Name),
		ttr.UsingTask(taskName),
	}, opts...)...)
}
