# Buildroot

To build the HAL for Linux, you need to install [Buildroot](https://buildroot.org/). Checkout [README.md](./linux/README.md) for more information.

## To run using qemu

After following the steps in [README.md](./linux/README.md), you will have bzImage and rootfs.cpio.gz files.

Next we need to create a filesystem image. We will use `mkfs.ext4` to create the filesystem image.

```bash
dd if=/dev/zero of=rootfs.img bs=1M count=10240
mkfs.ext4 ./rootfs.img
```

Now we can run the QEMU VM with the filesystem image.

```bash
bash buildroot/qemu.sh
```

Login to the VM using the following credentials:

- Username: `root`

To mount the filesystem image, you can use the following command:

```bash
mkdir -p /mnt/docker-volume
mount /dev/vda /mnt/docker-volume
```

You can now access the persitent storage of the VM using the following command:

```bash
ls /mnt/docker-volume
```
