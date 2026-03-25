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
  name     = "e2etest-terraform-{{ $sys.id }}-queue"
  location = "{{ $sys.deploymentCell.region }}"
}

# Create a Service Bus Namespace
resource "azurerm_servicebus_namespace" "example" {
  name                = "e2etest-sb-namespace-{{ $sys.id }}"
  location            = azurerm_resource_group.e2e_rg.location
  resource_group_name = azurerm_resource_group.e2e_rg.name
  sku                 = "Standard"

  tags = {
    environment = "e2e"
  }
}

# Create a Service Bus Queue
resource "azurerm_servicebus_queue" "example" {
  name         = "e2etest-sb-{{ $sys.id }}"
  namespace_id = azurerm_servicebus_namespace.example.id

  max_size_in_megabytes = 1024
  default_message_ttl   = "P1D"  # 1 day

  # Optional: Dead lettering
  dead_lettering_on_message_expiration = true

  # Optional: Session support
  requires_session = false

  # Optional: Duplicate detection
  requires_duplicate_detection = true
  duplicate_detection_history_time_window = "PT10M"  # 10 minutes
}

# Output the primary connection string
output "servicebus_connection_string" {
  value     = azurerm_servicebus_namespace.example.default_primary_connection_string
  sensitive = true
}

# Output the queue name
output "pubsub_id" {
  value = azurerm_servicebus_queue.example.name
}

# Outputs that kustomize expects
output "redis_endpoint" {
  value = ""
}