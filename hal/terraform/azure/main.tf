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
  subscription_id = var.subscription_id
}

variable "subscription_id" {
  type        = string
  description = "Azure subscription ID"
}

variable "resource_group_name" {
  type        = string
  description = "Name of the resource group"
}

variable "location" {
  type        = string
  default     = "westus"
  description = "Azure region"
}

variable "vm_name" {
  type        = string
  description = "Name of the VM"
}

variable "machine_type" {
  type        = string
  default     = "Standard_DC2as_v5"
  description = "Azure VM size (Confidential VM)"
}

variable "cloud_init_config" {
  type        = string
  description = "Path to cloud-init configuration file"
}

variable "admin_username" {
  type    = string
  default = "cubeadmin"
}

variable "admin_password" {
  type      = string
  sensitive = true
  default   = "CubeAdmin123!"
}

data "cloudinit_config" "cube_agent" {
  gzip          = true
  base64_encode = true

  part {
    content_type = "text/cloud-config"
    content      = file(var.cloud_init_config)
    filename     = "cloud-config.yml"
  }
}

resource "azurerm_public_ip" "cube_pip" {
  name                = "${var.vm_name}-pip"
  location            = var.location
  resource_group_name = var.resource_group_name
  allocation_method   = "Static"
  sku                 = "Standard"

  tags = {
    app = "cube-ai"
  }
}

resource "azurerm_network_security_group" "cube_nsg" {
  name                = "${var.vm_name}-nsg"
  location            = var.location
  resource_group_name = var.resource_group_name

  security_rule {
    name                       = "AllowSSH"
    priority                   = 1000
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "AllowCubeAgent"
    priority                   = 1001
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "7001"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  tags = {
    app = "cube-ai"
  }
}

resource "azurerm_virtual_network" "cube_vnet" {
  name                = "${var.vm_name}-vnet"
  address_space       = ["10.0.0.0/16"]
  location            = var.location
  resource_group_name = var.resource_group_name

  tags = {
    app = "cube-ai"
  }
}

resource "azurerm_subnet" "cube_subnet" {
  name                 = "${var.vm_name}-subnet"
  resource_group_name  = var.resource_group_name
  virtual_network_name = azurerm_virtual_network.cube_vnet.name
  address_prefixes     = ["10.0.2.0/24"]
}

resource "azurerm_network_interface" "cube_nic" {
  name                = "${var.vm_name}-nic"
  location            = var.location
  resource_group_name = var.resource_group_name

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.cube_subnet.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.cube_pip.id
  }

  tags = {
    app = "cube-ai"
  }
}

resource "azurerm_network_interface_security_group_association" "cube_nsg_assoc" {
  network_interface_id      = azurerm_network_interface.cube_nic.id
  network_security_group_id = azurerm_network_security_group.cube_nsg.id
}

resource "azurerm_linux_virtual_machine" "cube_cvm" {
  name                = var.vm_name
  location            = var.location
  resource_group_name = var.resource_group_name
  size                = var.machine_type
  admin_username      = var.admin_username
  admin_password      = var.admin_password

  disable_password_authentication = false
  network_interface_ids           = [azurerm_network_interface.cube_nic.id]

  # Enable confidential VM features
  vtpm_enabled        = true
  secure_boot_enabled = true

  os_disk {
    name                     = "${var.vm_name}-osdisk"
    caching                  = "ReadWrite"
    storage_account_type     = "Premium_LRS"
    security_encryption_type = "VMGuestStateOnly"
  }

  # Use Ubuntu 24.04 LTS CVM image
  source_image_reference {
    publisher = "Canonical"
    offer     = "ubuntu-24_04-lts"
    sku       = "cvm"
    version   = "latest"
  }

  # Inject cloud-init
  custom_data = data.cloudinit_config.cube_agent.rendered

  tags = {
    app  = "cube-ai"
    type = "confidential-vm"
  }
}

output "vm_name" {
  value       = azurerm_linux_virtual_machine.cube_cvm.name
  description = "Name of the created VM"
}

output "vm_public_ip" {
  value       = azurerm_public_ip.cube_pip.ip_address
  description = "Public IP address of the VM"
}

output "ssh_command" {
  value       = "ssh ${var.admin_username}@${azurerm_public_ip.cube_pip.ip_address}"
  description = "Command to SSH into the VM"
}

output "cube_agent_url" {
  value       = "http://${azurerm_public_ip.cube_pip.ip_address}:7001"
  description = "Cube Agent URL"
}

output "admin_username" {
  value       = var.admin_username
  description = "Admin username"
}
