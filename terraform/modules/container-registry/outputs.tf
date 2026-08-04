output "login_server" {
  description = "The login server URL of the ACR"
  value       = azurerm_container_registry.this.login_server
}

output "id" {
  description = "The resource ID of the ACR"
  value       = azurerm_container_registry.this.id
}
