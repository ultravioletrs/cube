# Ubuntu Cloud-Init for Cube AI

This directory contains cloud-init configurations and QEMU launch scripts for running Cube AI in Ubuntu-based Confidential VMs (CVMs).

## Overview

Ubuntu 24.04 (Noble) has built-in support for both Intel TDX and AMD SEV-SNP confidential computing technologies. No additional kernel modules or packages need to be installed - the guest support is enabled by default in the kernel.

## Files

- `qemu.sh` - Main QEMU launch script with TDX/SNP support
- `user-data-tdx.yaml` - Cloud-init configuration for Intel TDX VMs
- `user-data-snp.yaml` - Cloud-init configuration for AMD SEV-SNP VMs
- `user-data-base.yaml` - Base configuration template (reference only)

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

# AMD SEV-SNP
sudo ./qemu.sh start_snp

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

- Ubuntu 24.04 kernel has `CONFIG_SEV_GUEST=y` enabled by default
- Guest attestation available via `/dev/sev-guest`
- Modules: `sev-guest`, `ccp` (loaded automatically)

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
- Host kernel with SEV-SNP support
- `/dev/sev` device available

### Common Requirements
- QEMU with confidential computing support
- OVMF firmware (for UEFI boot)
- KVM enabled
