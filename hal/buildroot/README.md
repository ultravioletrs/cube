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

By default the docker images have been pulled from docker registry and the the docker composition has been started. The folder which contains the docker compose file is at `/mnt/docker/cube/docker`. To see the running containers, run the following command:

```bash
docker ps -a
```

For local development, replace the following IP address entries in `docker/.env` with the IP address of the qemu virtual machine as follows:

```bash
 UV_CUBE_NEXTAUTH_URL=http://<ip-address>:${UI_PORT}
```
