# Ubuntu Cloud-Init for Cube AI

This directory contains cloud-init configurations and QEMU launch scripts for running Cube AI in Ubuntu-based Confidential VMs (CVMs).

## Overview

Ubuntu 24.04 (Noble) has built-in support for Intel TDX confidential computing. For AMD SEV-SNP, a custom kernel is required because the host uses SVSM/Coconut — the standard Ubuntu 24.04 kernel does not support it.

## Files

- `qemu.sh` - Main QEMU launch script with TDX/SNP support
- `user-data-tdx.yaml` - Cloud-init configuration for Intel TDX VMs
- `user-data-snp.yaml` - Cloud-init configuration for AMD SEV-SNP VMs
- `user-data-regular.yaml` - Cloud-init configuration for regular (non-CVM) VMs
- `debs/` - (Optional) Directory for custom kernel `.deb` packages, required only when the SNP host runs Coconut SVSM (see [SNP Kernel](#snp-custom-kernel))

## Usage

### Auto-detect CVM Support

```bash
sudo ./qemu.sh start
```

This will automatically detect available CVM support (TDX or SNP) and launch the VM with the appropriate configuration.

### Force Specific CVM Mode

```bash
# Intel TDX
sudo ./qemu.sh start_tdx

# AMD SEV-SNP (two steps required)
sudo ./qemu.sh prepare_snp   # Run once: installs custom kernel via cloud-init (regular KVM)
sudo ./qemu.sh start_snp     # Boot the prepared image with IGVM/SNP

# Regular KVM (no CVM)
sudo ./qemu.sh start_regular
```

### Environment Variables

```bash
# Force specific CVM mode
ENABLE_CVM=tdx sudo ./qemu.sh start
ENABLE_CVM=snp sudo ./qemu.sh start
ENABLE_CVM=none sudo ./qemu.sh start

# Customize VM resources
RAM=32768M CPU=16 sudo ./qemu.sh start

# Override IGVM file path (SNP only)
IGVM=/path/to/coconut-qemu.igvm sudo ./qemu.sh start_snp
```

### Detect Available Support

```bash
sudo ./qemu.sh detect
```

## CVM Support Details

### Intel TDX (Trust Domain Extensions)

- Ubuntu 24.04 kernel has `CONFIG_INTEL_TDX_GUEST=y` enabled by default
- Guest attestation available via `/sys/firmware/tdx` or configfs
- Quote generation via vsock (CID=2, port=4050)

### AMD SEV-SNP (Secure Nested Paging)

- Boots via IGVM using the Coconut SVSM QEMU (`/home/cocosai/bin/qemu-svsm/bin/qemu-system-x86_64`)
- Requires an IGVM file (default: `/etc/cocos/coconut-qemu.igvm`)
- If the host runs Coconut SVSM, a custom kernel is required (see [SNP Kernel](#snp-custom-kernel))
- Guest attestation available via `/dev/sev-guest`
- Modules: `sev-guest`, `ccp` (loaded automatically)
- Disk is automatically resized on first boot via `growpart`

### SNP Custom Kernel

A custom kernel is only required when the SNP host runs Coconut SVSM. The standard Ubuntu 24.04 kernel does not support Coconut SVSM, so a kernel built with SVSM support must be bundled into the seed image as `.deb` packages.

**Kernel requirements:**

The custom kernel must be built with the following options:
- `CONFIG_AMD_MEM_ENCRYPT=y` — AMD memory encryption support
- `CONFIG_SEV_GUEST=y` — SEV guest driver
- `CONFIG_TCG_PLATFORM=y` — required for vTPM support
- Coconut SVSM guest support patches applied

The kernel must be packaged as `.deb` files.

**Installing the kernel into the seed image:**

Place the `.deb` files in a `debs/` directory next to `qemu.sh`:

```
hal/ubuntu/
  qemu.sh
  user-data-snp.yaml
  debs/
    linux-image-*.deb
    linux-headers-*.deb
    linux-modules-*.deb  (if needed)
```

The `prepare_snp` command will automatically detect `debs/` and package them along with `user-data` and `meta-data` into the seed ISO using `genisoimage`:

```bash
genisoimage -output seed.img -volid cidata -rock hal/ubuntu/
```

The `hal/ubuntu/` directory must contain:
- `user-data` — the cloud-init configuration (copied from `user-data-snp.yaml`)
- `meta-data` — the VM instance metadata
- `debs/` — the custom kernel `.deb` packages

On first boot, cloud-init mounts the seed ISO, installs the `.deb` packages, runs `update-grub`, and the VM boots the new SNP-compatible kernel on next start.

**Dependency:**

```bash
sudo apt-get install genisoimage
```

## After First Boot

For local development, update the following in `docker/.env`:

```bash
UV_CUBE_NEXTAUTH_URL=http://<vm-ip-address>:${UI_PORT}
```

Default SSH access:
- **Port**: 6190 (forwarded from guest port 22)
- **User**: ultraviolet
- **Password**: password

## Host Requirements

### For TDX VMs
- Intel CPU with TDX support (4th Gen Xeon Scalable or newer)
- TDX-enabled BIOS/firmware
- Host kernel with TDX module initialized

### For SNP VMs
- AMD EPYC CPU with SEV-SNP support (Milan or newer)
- SEV-SNP enabled in BIOS
- Host kernel with SEV-SNP/SVSM support
- `/dev/sev` device available
- Coconut SVSM QEMU binary at `/home/cocosai/bin/qemu-svsm/bin/qemu-system-x86_64`
- IGVM file at `/etc/cocos/coconut-qemu.igvm` (or set `IGVM` env var)
- `genisoimage` installed (`apt-get install genisoimage`)
- Custom kernel `.deb` files in `debs/` (see [SNP Kernel](#snp-custom-kernel))

### Common Requirements
- QEMU with confidential computing support
- OVMF firmware (for UEFI boot)
- KVM enabled
