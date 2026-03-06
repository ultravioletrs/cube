# Cloud Deployment for Cube AI

This directory contains cloud-init configurations for deploying Cube AI on confidential VMs on GCP and Azure, using the [cocos-infra](https://github.com/ultravioletrs/cocos-infra) Terraform templates.

## Files

| File | Backend | Description |
|------|---------|-------------|
| `cube-agent-config.yml` | Ollama | Default configuration with Ollama for model management |
| `cube-agent-vllm-config.yml` | vLLM | High-performance configuration with vLLM |

## Prerequisites

- [OpenTofu](https://opentofu.org) or Terraform installed
- GCP or Azure account with permissions to create confidential VMs
- cocos-infra repository cloned:

```bash
git clone https://github.com/ultravioletrs/cocos-infra.git
cd cocos-infra
```

## Deployment

### Google Cloud Platform (GCP)

#### Step 1: Deploy KMS (for disk encryption)

```bash
cd gcp/kms
tofu init
tofu apply -var-file="../../terraform.tfvars"
```

Note the `disk_encryption_id` output — you'll need it in the next step.

#### Step 2: Configure terraform.tfvars

Create or update `terraform.tfvars` at the root of cocos-infra:

```hcl
vm_name                    = "cube-ai-vm"
project_id                 = "your-gcp-project-id"
region                     = "us-central1"
min_cpu_platform           = "AMD Milan"
confidential_instance_type = "SEV_SNP"
machine_type               = "n2d-standard-4"
vm_id                      = "cube-vm-001"
workspace_id               = "cube-workspace-001"
disk_encryption_id         = ""  # output from gcp/kms step above
cloud_init_config          = "/path/to/cube/hal/ubuntu/cloud/cube-agent-config.yml"
```

#### Step 3: Deploy the VM

```bash
cd gcp
tofu init
tofu apply -var-file="../terraform.tfvars"
```

---

### Microsoft Azure

#### Step 1: Authenticate

```bash
az login
```

#### Step 2: Deploy KMS (for disk encryption)

```bash
cd azure/kms
tofu init
tofu apply -var-file="../../terraform.tfvars"
```

Note the `disk_encryption_id` output.

#### Step 3: Configure terraform.tfvars

```hcl
vm_name             = "cube-ai-vm"
resource_group_name = "cube-ai-rg"
location            = "westus"
subscription_id     = "your-subscription-id"
machine_type        = "Standard_DC4ads_v5"
vm_id               = "cube-vm-001"
workspace_id        = "cube-workspace-001"
disk_encryption_id  = ""  # output from azure/kms step above
cloud_init_config   = "/path/to/cube/hal/ubuntu/cloud/cube-agent-config.yml"
```

#### Step 4: Deploy the VM

```bash
cd azure
tofu init
tofu apply -var-file="../terraform.tfvars"
```

## Configuration

### Choosing a Backend

- Use `cube-agent-config.yml` for Ollama — good for multi-model setups and general use
- Use `cube-agent-vllm-config.yml` for vLLM — OpenAI-compatible API, better throughput for single-model production use

### Environment Variables

Set these in `vllm.env` or `agent.env` inside the VM to customize:

| Variable | Default | Description |
|----------|---------|-------------|
| `CUBE_VLLM_MODEL` | `meta-llama/Llama-2-7b-hf` | HuggingFace model ID for vLLM |
| `UV_CUBE_AGENT_LOG_LEVEL` | `info` | Agent log level |
| `UV_CUBE_AGENT_CA_URL` | `` | Attestation Manager URL |

## After Deployment

SSH into the VM and check service status:

```bash
# Check cloud-init completed
cloud-init status --wait

# Check services
sudo systemctl status cube-agent
sudo systemctl status ollama   # or vllm

# Test agent health
curl http://localhost:7001/health
```
