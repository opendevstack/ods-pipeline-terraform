terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.2.0"
    }
  }
}

resource "tfcoremock_simple_resource" "example" {
  id      = "my-simple-resource"
  bool    = true
  number  = 42
  string  = var.hello
  float   = 42.23
  integer = 11
}