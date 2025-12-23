# Ubuntu Cloud-Init Configuration

This directory contains the cloud-init configuration file for deploying Cube Agent and Ollama on Ubuntu-based confidential VMs.

## Files

- `cube-agent-config.yml`: Cloud-init config that installs and starts:
  - Ollama (`ollama.service`, port `11434`)
  - Cube Agent (`cube-agent.service`, port `7001`)

## What the cloud-init config does

On first boot, `cube-agent-config.yml`:

- Installs dependencies (`curl`, `git`, `golang-go`, `build-essential`, ...)
- Creates users:
  - `cubeadmin` (sudo)
  - `ollama` (system)
- Writes:
  - `/etc/cube/agent.env` (Cube Agent runtime config)
  - `/etc/systemd/system/ollama.service`
  - `/etc/systemd/system/cube-agent.service`
  - `/usr/local/bin/pull-ollama-models.sh` (downloads models)
- Installs Ollama and starts it
- Pulls the default model (`tinyllama:1.1b`) in the background (log: `/var/log/cube/ollama-pull.log`)
- Clones this repo and builds `cube-agent`, installs it to `/usr/local/bin/cube-agent`, then starts it

## Customization

Edit `cube-agent-config.yml`:

- **Cube Agent settings**: change `/etc/cube/agent.env` (e.g. `UV_CUBE_AGENT_INSTANCE_ID`, `UV_CUBE_AGENT_TARGET_URL`, ports)
- **TLS/mTLS certificates**: uncomment certificate sections and add your certs
- **Ollama settings**: change `OLLAMA_HOST` in `ollama.service`
- **Models**: update `/usr/local/bin/pull-ollama-models.sh` or set `CUBE_MODELS` environment variable
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
