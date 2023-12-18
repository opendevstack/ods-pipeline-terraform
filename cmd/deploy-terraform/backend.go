package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const templateBackendKubernetesName = "backend-kubernetes.tf"

var (
	//go:embed backend-kubernetes.tf
	templateBackendKubernetesFS embed.FS
	templateBackendKubernetes   *template.Template
)

func init() {
	templateBackendKubernetes = template.Must(template.New(templateBackendKubernetesName).ParseFS(templateBackendKubernetesFS, templateBackendKubernetesName))
}

type BackendKubernetesData struct {
	SecretSuffix string
}

func (d *deployTerraform) renderBackend() error {
	destination := filepath.Join(d.opts.terraformDir, "backend-kubernetes.tf")
	w, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destination, err)
	}
	if _, err := w.Write(
		[]byte("// File is generated; DO NOT EDIT.\n\n"),
	); err != nil {
		return err
	}
	err = templateBackendKubernetes.ExecuteTemplate(w, templateBackendKubernetesName, &BackendKubernetesData{
		SecretSuffix: fmt.Sprintf("%s-%s", d.ctxt.Component, d.opts.targetEnvironment),
	})
	if err != nil {
		return fmt.Errorf("rendering internal kubernetes backend template failed: %w", err)
	}
	return nil
}
