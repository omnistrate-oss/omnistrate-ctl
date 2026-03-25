terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }
}

provider "azurerm" {
  features {}
}

# Create a resource group
resource "azurerm_resource_group" "e2e_rg" {
  name     = "e2etest-terraform-{{ $sys.id }}"
  location = "{{ $sys.deploymentCell.region }}"
}

# Create a virtual network within the resource group
resource "azurerm_virtual_network" "e2e_network" {
  name                = "e2etest-terraform-network-{{ $sys.id }}"
  resource_group_name = azurerm_resource_group.e2e_rg.name
  location            = azurerm_resource_group.e2e_rg.location
  address_space       = ["10.0.0.0/16"]
}

output "resource_group_name" {
  value = azurerm_resource_group.e2e_rg.name
}

output "network_id" {
  value = azurerm_virtual_network.e2e_network.id
}

# Values that kustomize expects
output "db_endpoints_1" {
  value = ""
}

output "db_endpoints_2" {
  value = {
    endpoint: ""
  }
}
