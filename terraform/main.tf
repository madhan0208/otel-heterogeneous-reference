terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "otel" {
  name     = var.resource_group_name
  location = var.location
}

resource "azurerm_container_registry" "otel" {
  name                = var.acr_name
  resource_group_name = azurerm_resource_group.otel.name
  location            = azurerm_resource_group.otel.location
  sku                 = "Basic"
  admin_enabled       = false
}
