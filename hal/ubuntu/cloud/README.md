# Cloud-Init Configuration for Cube AI

This directory contains cloud-init configuration files for deploying Cube AI on Ubuntu-based confidential virtual machines (CVMs) on cloud providers.

## Cloud-Init Files

| File | Backend | Description |
|------|---------|-------------|
| `cube-agent-config.yml` | Ollama | Default configuration with Ollama for easy model management |
| `cube-agent-vllm-config.yml` | vLLM | High-performance configuration with vLLM for production workloads |

## Choosing a Backend

### Ollama (Recommended for Getting Started)

Use `cube-agent-config.yml` for:

- Quick setup and experimentation
- Running multiple models
- CPU or small GPU deployments
- Built-in quantization support (Q4_0, Q4_1, Q8_0)

### vLLM (Recommended for Production)

Use `cube-agent-vllm-config.yml` for:

- Maximum inference throughput
- Large-scale production deployments
- Multi-GPU setups with tensor parallelism
- Continuous batching and PagedAttention

## Deployment

### Google Cloud Platform (GCP)

```bash
# Clone infrastructure templates
git clone https://github.com/ultravioletrs/cocos-infra.git
cd cocos-infra

# Configure terraform.tfvars
cat >> terraform.tfvars << 'EOF'
vm_name = "cube-ai-vm"
project_id = "your-gcp-project-id"
region = "us-central1"
zone = "us-central1-a"
min_cpu_platform = "AMD Milan"
confidential_instance_type = "SEV_SNP"
machine_type = "n2d-standard-4"
cloud_init_config = "/path/to/cube/hal/ubuntu/cloud/cube-agent-config.yml"
EOF

# Deploy
cd gcp
tofu init && tofu apply -var-file="../terraform.tfvars"
```

### Microsoft Azure

```bash
# Configure terraform.tfvars
cat >> terraform.tfvars << 'EOF'
vm_name = "cube-ai-vm"
resource_group_name = "cube-ai-rg"
location = "westus"
subscription_id = "your-subscription-id"
machine_type = "Standard_DC4ads_v5"
cloud_init_config = "/path/to/cube/hal/ubuntu/cloud/cube-agent-config.yml"
EOF

# Deploy
cd azure
az login
tofu init && tofu apply -var-file="../terraform.tfvars"
```

## Configuration

### Environment Variables

Set these environment variables before deployment to customize the configuration:

| Variable | Default | Description |
|----------|---------|-------------|
| `CUBE_MODELS` | `tinyllama:1.1b` | Comma-separated Ollama models to pull |
| `CUBE_VLLM_MODEL` | `meta-llama/Llama-2-7b-hf` | HuggingFace model ID for vLLM |
| `CUBE_VLLM_GPU_COUNT` | `1` | Number of GPUs for tensor parallelism |
| `CUBE_AGENT_VERSION` | `latest` | Cube Agent release version |
| `HF_TOKEN` | - | HuggingFace token for gated models |

### TLS/mTLS Certificates

For production deployments, replace the self-signed certificate generation with your own certificates:

1. Edit the cloud-init file
2. Uncomment the certificate file sections
3. Replace placeholder content with your certificates
4. Update `/etc/cube/agent.env` to enable TLS

### Custom Models (Ollama)

Pull additional models by setting `CUBE_MODELS`:

```bash
export CUBE_MODELS="llama2:7b,mistral:latest,codellama:13b"
```

Or create a custom Modelfile after deployment:

```bash
ssh cubeadmin@<vm-ip>
cat > /tmp/Modelfile << 'EOF'
FROM llama2:7b
PARAMETER temperature 0.7
SYSTEM You are a helpful AI assistant.
EOF
sudo -u ollama /usr/local/bin/ollama create custom-assistant -f /tmp/Modelfile
```

## Verification

After deployment, verify the services are running:

```bash
# Check cloud-init completion
ssh cubeadmin@<vm-ip>
cloud-init status --wait

# Check service status
sudo systemctl status cube-agent
sudo systemctl status ollama  # or vllm

# Test health endpoint
curl http://localhost:7001/health

# Test chat completion
curl http://<vm-ip>:7001/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tinyllama:1.1b",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## VM Size Recommendations

### GCP

| Use Case | Machine Type | vCPUs | RAM |
|----------|--------------|-------|-----|
| Development | `n2d-standard-2` | 2 | 8GB |
| Production (Ollama) | `n2d-standard-4` | 4 | 16GB |
| Production (vLLM) | `n2d-standard-8` | 8 | 32GB |
| Production (vLLM + GPU) | `n1-standard-8` + T4 | 8 | 30GB |

### Azure

| Use Case | Machine Type | vCPUs | RAM |
|----------|--------------|-------|-----|
| Development | `Standard_DC2ads_v5` | 2 | 8GB |
| Production (Ollama) | `Standard_DC4ads_v5` | 4 | 16GB |
| Production (vLLM) | `Standard_DC8ads_v5` | 8 | 32GB |
| Production (vLLM + GPU) | `Standard_NC6s_v3` | 6 | 112GB |

## Troubleshooting

### Cloud-init not completing

```bash
# Check cloud-init logs
sudo cat /var/log/cloud-init-output.log
sudo cat /var/log/cloud-init.log
```

### Cube Agent not starting

```bash
# Check service logs
sudo journalctl -u cube-agent -f

# Verify configuration
cat /etc/cube/agent.env
```

### Ollama not responding

```bash
# Check service logs
sudo journalctl -u ollama -f

# Check if models are downloaded
sudo -u ollama /usr/local/bin/ollama list
```

### vLLM GPU issues

```bash
# Check NVIDIA driver
nvidia-smi

# Check vLLM logs
sudo journalctl -u vllm -f
```
