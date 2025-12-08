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
	-append "console=ttyS0" \
	-object memory-backend-memfd,id=mem0,size=20G \
	-initrd /home/sammy/cube-cvm/rootfs.cpio.gz \
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
/etc/init.d/S95agent status
```

View the process:

```bash
ps aux | grep cube-agent
```

Restart the service:

```bash
/etc/init.d/S95agent restart
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
