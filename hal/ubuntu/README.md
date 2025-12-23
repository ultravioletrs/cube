# Ubuntu Cloud-Init Configuration

This directory contains the cloud-init configuration file for deploying Cube Agent and Ollama on Ubuntu-based confidential VMs.

## Files

- `cube-agent-config.yml`: Cloud-init config that installs and starts:
  - Ollama (`ollama.service`, port `11434`) OR vLLM (`vllm.service`, port `8000`)
  - Cube Agent (`cube-agent.service`, port `7001`)

## What the cloud-init config does

On first boot, `cube-agent-config.yml`:

- Installs dependencies (`curl`, `git`, `golang-go`, `build-essential`, `python3`, `python3-pip`, ...)
- Creates users:
  - `cubeadmin` (sudo)
  - `ollama` (system - used for both Ollama and vLLM)
- Writes systemd service files:
  - `/etc/systemd/system/ollama.service`
  - `/etc/systemd/system/vllm.service`
  - `/etc/systemd/system/cube-agent.service`
  - `/usr/local/bin/pull-ollama-models.sh` (downloads Ollama models)
- **Conditionally installs AI backend** based on `CUBE_AI_BACKEND` environment variable:
  - **Ollama** (default): Installs Ollama, starts service, pulls models in background
  - **vLLM**: Installs vLLM via pip, starts service with specified model
- Configures `/etc/cube/agent.env` with appropriate backend URL
- Clones this repo and builds `cube-agent`, installs it to `/usr/local/bin/cube-agent`, then starts it

## Customization

Edit `cube-agent-config.yml`:

- **AI Backend selection**: Set `CUBE_AI_BACKEND` environment variable before deployment
  - `CUBE_AI_BACKEND=ollama` (default) - Uses Ollama for model inference
  - `CUBE_AI_BACKEND=vllm` - Uses vLLM for model inference
- **Cube Agent settings**: change `/etc/cube/agent.env.template` (e.g. `UV_CUBE_AGENT_INSTANCE_ID`, ports)
- **TLS/mTLS certificates**: uncomment certificate sections and add your certs
- **Ollama models**: set `CUBE_MODELS` environment variable (e.g. `CUBE_MODELS="llama2:7b,mistral:latest"`)
- **vLLM model**: set `CUBE_VLLM_MODEL` environment variable (e.g. `CUBE_VLLM_MODEL="meta-llama/Llama-2-7b-hf"`)
- **Source build**: update the `git clone` URL/branch if you want to build from a fork/branch

## Logs

- Cloud-init: `/var/log/cloud-init-output.log`
- Model pulls: `/var/log/cube/ollama-pull.log`
- Setup completion: `/var/log/cube/setup.log`
- Services: `journalctl -u ollama -u cube-agent`

## Testing

### 1) Validate the config (local)

From the repo root:

```bash
cloud-init schema --config-file hal/ubuntu/cube-agent-config.yml
```

### 2) Deploy on a cloud VM (Terraform/OpenTofu)

Use Terraform/OpenTofu templates from the [cocos-infra](https://github.com/ultravioletrs/cocos-infra) repository and reference this cloud-init file via the `cloud_init_config` variable.

```bash
# Clone cocos-infra
git clone https://github.com/ultravioletrs/cocos-infra.git

# For GCP, set cloud_init_config in terraform.tfvars
cd cocos-infra/gcp
cat > terraform.tfvars <<EOF
cloud_init_config = "/path/to/cube/hal/ubuntu/cube-agent-config.yml"
# ... other variables
EOF

terraform init
terraform apply
```

**Selecting AI Backend:**

By default, Ollama is used. To deploy with vLLM instead:

```bash
# Option 1: Set environment variable before apply
export CUBE_AI_BACKEND=vllm
export CUBE_VLLM_MODEL="meta-llama/Llama-2-7b-hf"  # Optional: specify model
terraform apply

# Option 2: Modify cloud-init file to set default
# Edit hal/ubuntu/cube-agent-config.yml and change:
# - export AI_BACKEND="${CUBE_AI_BACKEND:-vllm}"
```

**Backend comparison:**
- **Ollama** (default): Easy to use, supports multiple models, better for CPU/small GPU
  - Port: 11434
  - Models pulled automatically (tinyllama:1.1b by default)
- **vLLM**: High-performance inference, requires GPU, optimized for large models
  - Port: 8000
  - Requires specifying model (facebook/opt-125m by default for testing)

**Note:** The cocos-infra firewall rules allow port 7002 (CoCoS default). To access Cube Agent on port 7001, you'll need to add a firewall rule:

```bash
# After terraform apply, add firewall rule for Cube Agent
gcloud compute firewall-rules create allow-cube-agent-7001 \
  --allow=tcp:7001 \
  --source-ranges=0.0.0.0/0 \
  --target-tags=cube-ai-cvm-01 \
  --project=valued-base-354714
```

After `apply`, SSH into the VM and run:

```bash
cloud-init status --wait
sudo systemctl status ollama cube-agent --no-pager
curl -sf http://localhost:7001/health
```

If you need to hit the agent from outside the VM, ensure your firewall/security group allows inbound TCP `7001`.

## After the first boot (local development)

For local development, replace the following IP address entries in `docker/.env` with the IP address of the qemu virtual machine as follows:

```bash
UV_CUBE_NEXTAUTH_URL=http://<ip-address>:${UI_PORT}
```
