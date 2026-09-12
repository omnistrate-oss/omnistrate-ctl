terraform {
  required_version = "= 1.12.2"

  # These constraints are copied from the Pavo BYOC 925.0 rendered module.
  # The committed lock file selects the same versions observed in Pavo's
  # retained Terraform executor workspace.
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.33.0, < 6.0"
    }

    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.30"
    }

    kubectl = {
      source  = "alekc/kubectl"
      version = ">= 2.2.0, < 3.0"
    }

    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.13"
    }

    ec = {
      source  = "elastic/ec"
      version = "~> 0.12"
    }

    elasticstack = {
      source  = "elastic/elasticstack"
      version = "~> 0.15"
    }

    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }

    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }

    time = {
      source  = "hashicorp/time"
      version = "~> 0.11"
    }
  }
}

variable "aws_region" {
  description = "Region supplied by the Omnistrate deployment cell."
  type        = string
}

provider "aws" {
  region = var.aws_region
}

output "pavo_provider_mirror_validation" {
  description = "Exact Pavo provider versions exercised by this disposable mirror POC."
  value = {
    aws          = "5.100.0"
    kubernetes   = "2.38.0"
    kubectl      = "2.4.1"
    helm         = "2.17.0"
    ec           = "0.13.0"
    elasticstack = "0.16.4"
    tls          = "4.4.1"
    random       = "3.9.1"
    time         = "0.14.2"
  }
}
