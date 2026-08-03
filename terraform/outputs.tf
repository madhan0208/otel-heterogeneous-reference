output "resource_group_name" {
  value = azurerm_resource_group.otel.name
}

output "acr_login_server" {
  value = azurerm_container_registry.otel.login_server
}
