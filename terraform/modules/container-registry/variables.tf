variable "name" {
  description = "Name of the container registry"
  type        = string
}

variable "resource_group_name" {
  description = "Resource group to deploy into"
  type        = string
}

variable "location" {
  description = "Azure region"
  type        = string
}

variable "sku" {
  description = "SKU tier - Basic, Standard, or Premium"
  type        = string
  default     = "Basic"
}
