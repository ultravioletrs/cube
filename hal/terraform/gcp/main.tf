terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

variable "project_id" {
  type        = string
  description = "GCP Project ID"
}

variable "region" {
  type        = string
  default     = "us-central1"
  description = "GCP region"
}

variable "vm_name" {
  type        = string
  description = "Name of the VM"
}

variable "machine_type" {
  type        = string
  default     = "n2d-standard-2"
  description = "Machine type for the VM"
}

variable "min_cpu_platform" {
  type        = string
  default     = "AMD Milan"
  description = "Minimum CPU platform"
}

variable "confidential_instance_type" {
  type        = string
  default     = "SEV"
  description = "Type of confidential computing (SEV or SEV_SNP)"
}

variable "cloud_init_config" {
  type        = string
  description = "Path to cloud-init configuration file"
}

data "cloudinit_config" "cube_agent" {
  gzip          = false
  base64_encode = false

  part {
    content_type = "text/cloud-config"
    content      = file(var.cloud_init_config)
    filename     = "cloud-config.yml"
  }
}

resource "google_compute_instance" "cube_cvm" {
  name             = var.vm_name
  machine_type     = var.machine_type
  min_cpu_platform = var.min_cpu_platform
  zone             = "${var.region}-a"
  tags             = ["cube-cvm", var.vm_name]

  labels = {
    app  = "cube-ai"
    type = "confidential-vm"
  }

  scheduling {
    on_host_maintenance = "TERMINATE"
  }

  boot_disk {
    initialize_params {
      image = "projects/ubuntu-os-cloud/global/images/ubuntu-2404-noble-amd64-v20241219"
      size  = 50
      type  = "pd-balanced"
    }
  }

  network_interface {
    network = "default"
    access_config {
      // Ephemeral public IP
    }
  }

  confidential_instance_config {
    enable_confidential_compute = true
    confidential_instance_type  = var.confidential_instance_type
  }

  metadata = {
    user-data = data.cloudinit_config.cube_agent.rendered
  }

  shielded_instance_config {
    enable_integrity_monitoring = true
    enable_secure_boot          = true
    enable_vtpm                 = true
  }
}

resource "google_compute_firewall" "cube_agent" {
  name    = "allow-cube-agent-${var.vm_name}"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["7001"]
  }

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["cube-cvm"]
}

output "vm_name" {
  value       = google_compute_instance.cube_cvm.name
  description = "Name of the created VM"
}

output "vm_public_ip" {
  value       = google_compute_instance.cube_cvm.network_interface.0.access_config.0.nat_ip
  description = "Public IP address of the VM"
}

output "vm_zone" {
  value       = google_compute_instance.cube_cvm.zone
  description = "Zone where the VM is deployed"
}

output "ssh_command" {
  value       = "gcloud compute ssh ${google_compute_instance.cube_cvm.name} --zone=${google_compute_instance.cube_cvm.zone}"
  description = "Command to SSH into the VM"
}

output "cube_agent_url" {
  value       = "http://${google_compute_instance.cube_cvm.network_interface.0.access_config.0.nat_ip}:7001"
  description = "Cube Agent URL"
}
