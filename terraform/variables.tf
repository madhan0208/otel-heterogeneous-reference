variable "location" {
  description = "Azure region for all resources"
  type        = string
  default     = "Germany West Central"
}

variable "resource_group_name" {
  description = "Name of the resource group"
  type        = string
  default     = "otel-thesis"
}

variable "acr_name" {
  description = "Name of the Azure Container Registry (must be globally unique)"
  type        = string
  default     = "madhanotelregistry"
}
