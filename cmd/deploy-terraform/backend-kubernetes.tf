terraform {
    backend "kubernetes" {
        secret_suffix = "{{.SecretSuffix}}"
        in_cluster_config = true 
    }
}