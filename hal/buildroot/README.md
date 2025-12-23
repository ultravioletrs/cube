# Buildroot

To build the HAL for Linux, you need to install [Buildroot](https://buildroot.org/). Checkout [README.md](./linux/README.md) for more information.

## To run using qemu

After following the steps in [README.md](./linux/README.md), you will have bzImage and rootfs.cpio.gz files.

Next we need to create a filesystem image. We will use `mkfs.ext4` to create the filesystem image. To do so, run the following command from `~/cube/hal/buildroot` directory:

```bash
dd if=/dev/zero of=rootfs.img bs=1M count=30720
mkfs.ext4 ./rootfs.img
```

Now we can run the QEMU VM with the filesystem image from `~/cube/hal` directory.

```bash
sudo bash buildroot/qemu.sh start_cvm
```

If you want to start a normal VM, you can run:

```bash
sudo bash buildroot/qemu.sh start
```

### Manual CVM Deployment

You can also manually deploy the CVM using the following QEMU command:

```bash
/usr/bin/qemu-system-x86_64 \
	-enable-kvm \
	-m 20G -smp cores=16,sockets=1,threads=1 \
	-cpu host \
	-object '{"qom-type":"tdx-guest","id":"tdx","quote-generation-socket":{"type": "vsock", "cid":"2","port":"4050"}}' \
	-machine q35,kernel_irqchip=split,confidential-guest-support=tdx,memory-backend=mem0,hpet=off \
	-bios /usr/share/ovmf/OVMF.fd \
	-nographic \
	-nodefaults \
	-no-reboot \
	-serial mon:stdio \
	-device virtio-net-pci,netdev=nic0_td \
	-netdev user,id=nic0_td,hostfwd=tcp::7021-:7002 \
	-kernel /home/sammy/cube-cvm/bzImage \
	-append "root=/dev/vda rw console=ttyS0" \
	-object memory-backend-memfd,id=mem0,size=20G \
	-drive file=/home/sammy/cube-cvm/rootfs.ext4,format=raw,if=virtio \
	-device vhost-vsock-pci,guest-cid=6 \
	-monitor pty \
	-monitor unix:monitor,server,nowait
```

Login to the VM using the following credentials:

- Username: `root`

Attest the VM by running the following command:

```bash
bash /cube/attest.sh
```

You will see a report similar to the following:

```bash
The AMD ARK was self-signed!
The AMD ASK was signed by the AMD ARK!
The VCEK was signed by the AMD ASK!
Reported TCB Boot Loader from certificate matches the attestation report.
Reported TCB TEE from certificate matches the attestation report.
Reported TCB SNP from certificate matches the attestation report.
Reported TCB Microcode from certificate matches the attestation report.
Chip ID from certificate matches the attestation report.
VEK signed the Attestation Report!
Measurement from SNP Attestation Report: daa2e216eafd8c6404b72157a130500ab0c0944064c8e1009ebf5e910371caf57a6711654108a01a69baaa1a05759cf0
```

## Verifying Cube Agent is Running

The Cube Agent is automatically started on boot. To verify it's running:

### For systemd-based systems

Check the service status:

```bash
systemctl status cube-agent
```

View the service logs:

```bash
journalctl -u cube-agent -f
```

Restart the service if needed:

```bash
systemctl restart cube-agent
```

### For SysV init systems

Check the service status:

```bash
/etc/init.d/S95cube-agent status
```

View the process:

```bash
ps aux | grep cube-agent
```

Restart the service:

```bash
/etc/init.d/S95cube-agent restart
```

### Test the Agent API

The agent runs on port `7001` by default. Test the endpoint:

```bash
curl http://localhost:7001/health
```

Check the agent configuration:

```bash
cat /etc/cube/agent.env
```

## Cloud-init provisioning (Terraform deployments)

For deploying Cube Agent on cloud CVMs, use the Terraform/OpenTofu templates from the [cocos-infra](https://github.com/ultravioletrs/cocos-infra) repository with Cube's cloud-init configuration located in [`hal/ubuntu/cube-agent-config.yml`](../ubuntu/cube-agent-config.yml).

### What the cloud-init config does

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

### Customization

Edit `hal/ubuntu/cube-agent-config.yml`:

- **Cube Agent settings**: change `/etc/cube/agent.env` (e.g. `UV_CUBE_AGENT_INSTANCE_ID`, `UV_CUBE_AGENT_TARGET_URL`, ports)
- **TLS/mTLS certificates**: uncomment certificate sections and add your certs
- **Ollama settings**: change `OLLAMA_HOST` in `ollama.service`
- **Models**: update `/usr/local/bin/pull-ollama-models.sh` or set `CUBE_MODELS` environment variable
- **Source build**: update the `git clone` URL/branch if you want to build from a fork/branch

### Logs

- Cloud-init: `/var/log/cloud-init-output.log`
- Model pulls: `/var/log/cube/ollama-pull.log`
- Setup completion: `/var/log/cube/setup.log`
- Services: `journalctl -u ollama -u cube-agent`

### Testing

#### 1) Validate the config (local)

From the repo root:

```bash
cloud-init schema --config-file hal/ubuntu/cube-agent-config.yml
```

#### 2) Deploy on a cloud VM (Terraform/OpenTofu)

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
