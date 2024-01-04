package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	ott "github.com/opendevstack/ods-pipeline/pkg/odstasktest"
	"github.com/opendevstack/ods-pipeline/pkg/pipelinectxt"
	ttr "github.com/opendevstack/ods-pipeline/pkg/tektontaskrun"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func TestPlanTerraformTask(t *testing.T) {
	k8sClient := newK8sClient(t)
	_, nsCleanup := createRoleBindingsOrFatal(t, k8sClient, namespaceConfig.Name)
	defer nsCleanup()
	_, secretCleanup := createSecretsOrFatal(t, k8sClient, namespaceConfig.Name, map[string]string{
		"TF_VAR_hello": "Hello ods-pipeline-terraform!",
	})
	defer secretCleanup()
	if err := runTask(
		ttr.WithStringParams(
			map[string]string{
				"plan-only": "true",
				"verbose":   "true",
			},
		),
		withWorkspace(t, "terraform-sample"),
		ttr.AfterRun(func(config *ttr.TaskRunConfig, run *tekton.TaskRun, logs bytes.Buffer) {
			dir := config.WorkspaceConfigs["source"].Dir
			fmt.Println(dir)
			ott.AssertFileContentContains(t,
				dir,
				filepath.Join(pipelinectxt.DeploymentsPath, fmt.Sprintf("plan-%s.txt", "dev")),
				"Terraform used the selected providers to generate the following execution",
				"plan. Resource actions are indicated with the following symbols:",
				"  + create",
				"",
				"Terraform will perform the following actions:",
				"",
				"  # tfcoremock_simple_resource.example will be created",
				"  + resource \"tfcoremock_simple_resource\" \"example\" {",
				"+ bool    = true",
				"+ float   = 42.23",
				"+ id      = \"my-simple-resource\"",
				"+ integer = 11",
				"+ number  = 42",
				"+ string  = \"Hello ods-pipeline-terraform!\"",
				"}",
				"",
				"Plan: 1 to add, 0 to change, 0 to destroy.",
			)
		}),
	); err != nil {
		t.Fatal(err)
	}
}

func TestApplyTerraformTask(t *testing.T) {
	k8sClient := newK8sClient(t)
	_, nsCleanup := createRoleBindingsOrFatal(t, k8sClient, namespaceConfig.Name)
	defer nsCleanup()
	_, secretCleanup := createSecretsOrFatal(t, k8sClient, namespaceConfig.Name, map[string]string{
		"TF_VAR_hello": "Hello ods-pipeline-terraform!",
	})
	defer secretCleanup()
	if err := runTask(
		ttr.WithStringParams(
			map[string]string{
				"verbose": "true",
			},
		),
		withWorkspace(t, "terraform-sample"),
		ttr.AfterRun(func(config *ttr.TaskRunConfig, run *tekton.TaskRun, logs bytes.Buffer) {
			dir := config.WorkspaceConfigs["source"].Dir
			fmt.Println(dir)
			ott.AssertFileContentContains(t,
				dir,
				filepath.Join(pipelinectxt.DeploymentsPath, fmt.Sprintf("plan-%s.txt", "dev")),
				"Terraform used the selected providers to generate the following execution",
				"plan. Resource actions are indicated with the following symbols:",
				"  + create",
				"",
				"Terraform will perform the following actions:",
				"",
				"  # tfcoremock_simple_resource.example will be created",
				"  + resource \"tfcoremock_simple_resource\" \"example\" {",
				"+ bool    = true",
				"+ float   = 42.23",
				"+ id      = \"my-simple-resource\"",
				"+ integer = 11",
				"+ number  = 42",
				"+ string  = \"Hello ods-pipeline-terraform!\"",
				"}",
				"",
				"Plan: 1 to add, 0 to change, 0 to destroy.",
			)
		}),
	); err != nil {
		t.Fatal(err)
	}
}
func TestApplyTerraformTaskWithDifferentDir(t *testing.T) {
	k8sClient := newK8sClient(t)
	_, nsCleanup := createRoleBindingsOrFatal(t, k8sClient, namespaceConfig.Name)
	defer nsCleanup()
	_, secretCleanup := createSecretsOrFatal(t, k8sClient, namespaceConfig.Name, map[string]string{
		"TF_VAR_hello": "Hello ods-pipeline-terraform!",
	})
	tfDir := "tf"
	defer secretCleanup()
	if err := runTask(
		ttr.WithStringParams(
			map[string]string{
				"terraform-dir": fmt.Sprintf("./%s", tfDir),
				"verbose":       "true",
			},
		),
		withWorkspace(t, "terraform-sample", func(c *ttr.WorkspaceConfig) error {
			renameTerraformDir(t, c.Dir, "terraform", tfDir)
			return nil
		}),
		ttr.AfterRun(func(config *ttr.TaskRunConfig, run *tekton.TaskRun, logs bytes.Buffer) {
			dir := config.WorkspaceConfigs["source"].Dir
			fmt.Println(dir)
			ott.AssertFileContentContains(t,
				dir,
				filepath.Join(pipelinectxt.DeploymentsPath, fmt.Sprintf("plan-%s.txt", "dev")),
				"Terraform used the selected providers to generate the following execution",
				"plan. Resource actions are indicated with the following symbols:",
				"  + create",
				"",
				"Terraform will perform the following actions:",
				"",
				"  # tfcoremock_simple_resource.example will be created",
				"  + resource \"tfcoremock_simple_resource\" \"example\" {",
				"+ bool    = true",
				"+ float   = 42.23",
				"+ id      = \"my-simple-resource\"",
				"+ integer = 11",
				"+ number  = 42",
				"+ string  = \"Hello ods-pipeline-terraform!\"",
				"}",
				"",
				"Plan: 1 to add, 0 to change, 0 to destroy.",
			)
		}),
	); err != nil {
		t.Fatal(err)
	}
}

func newK8sClient(t *testing.T) *kubernetes.Clientset {
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	kubernetesClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	return kubernetesClientset
}

func createRoleBindingsOrFatal(t *testing.T, clientset *kubernetes.Clientset, ctxtNamespace string) (rb *rbacv1.RoleBinding, cleanup func()) {
	rb, err := createRoleBindings(clientset, ctxtNamespace)
	if err != nil {
		t.Fatal(err)
	}
	return rb, func() {
		if err := clientset.RbacV1().RoleBindings(ctxtNamespace).Delete(context.TODO(), rb.Name, metav1.DeleteOptions{}); err != nil {
			t.Logf("Failed to delete role binding %s: %s", rb.Name, err)
		}
	}
}

func createRoleBindings(clientset *kubernetes.Clientset, ctxtNamespace string) (*rbacv1.RoleBinding, error) {
	rb, err := clientset.RbacV1().RoleBindings(ctxtNamespace).Create(
		context.Background(),
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline-terraform",
				Namespace: ctxtNamespace,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "pipeline",
					Namespace: ctxtNamespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "admin",
			},
		},
		metav1.CreateOptions{})

	return rb, err
}

func createSecretsOrFatal(t *testing.T, clientset *kubernetes.Clientset, ctxtNamespace string, vars map[string]string) (secret *v1.Secret, cleanup func()) {
	secret, err := createSecrets(clientset, ctxtNamespace, vars)
	if err != nil {
		t.Fatal(err)
	}
	return secret, func() {
		if err := clientset.CoreV1().Secrets(ctxtNamespace).Delete(context.TODO(), secret.Name, metav1.DeleteOptions{}); err != nil {
			t.Logf("Failed to delete secret %s: %s", secret.Name, err)
		}
	}
}

func createSecrets(clientset *kubernetes.Clientset, ctxtNamespace string, vars map[string]string) (*v1.Secret, error) {

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "terraform-envs-dev",
			Namespace: ctxtNamespace,
		},
		StringData: vars,
	}
	createdSecret, err := clientset.CoreV1().Secrets(ctxtNamespace).Create(
		context.Background(), secret, metav1.CreateOptions{})

	return createdSecret, err
}

func renameTerraformDir(t *testing.T, wsDir string, originalDir, targetDir string) {
	err := os.Rename(
		filepath.Join(wsDir, originalDir),
		filepath.Join(wsDir, targetDir),
	)
	if err != nil {
		t.Fatal(err)
	}
}

func withWorkspace(t *testing.T, dir string, opts ...ttr.WorkspaceOpt) ttr.TaskRunOpt {
	return ott.WithGitSourceWorkspace(
		t, filepath.Join("../testdata/workspaces", dir), namespaceConfig.Name,
		opts...,
	)
}
