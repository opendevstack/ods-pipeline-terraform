# ods-pipeline-terraform

[![Tests](https://github.com/opendevstack/ods-pipeline-terraform/actions/workflows/main.yaml/badge.svg)](https://github.com/opendevstack/ods-pipeline-terraform/actions/workflows/main.yaml)

Tekton task for use with [ODS Pipeline](https://github.com/opendevstack/ods-pipeline) to provision/configure infrastructure using Terraform .


## Usage

```yaml
tasks:
- name: build
  taskRef:
    resolver: git
    params:
    - { name: url, value: https://github.com/opendevstack/ods-pipeline-terraform.git }
    - { name: revision, value: v0.1.0 }
    - { name: pathInRepo, value: tasks/deploy.yaml }
    workspaces:
    - { name: source, workspace: shared-workspace }
```

See the [documentation](/docs/deploy.adoc) for details and available parameters.

## About this repository

`docs` and `tasks` are generated directories from recipes located in `build`. See the `Makefile` target for how everything fits together.
