# Ubuntu Cloud-Init for Cube AI

This directory contains cloud-init configurations and QEMU launch scripts for running Cube AI in Ubuntu-based Confidential VMs (CVMs).

## Directory Structure

```
hal/ubuntu/
  qemu.sh                      # QEMU launch script (local deployment)
  user-data-tdx.yaml           # Cloud-init for local QEMU TDX VMs (Ollama backend)
  user-data-snp.yaml           # Cloud-init for local QEMU SNP VMs (Ollama backend)
  user-data-regular.yaml       # Cloud-init for local QEMU regular VMs (Ollama backend)
  user-data-vllm-tdx.yaml     # Cloud-init for local QEMU TDX VMs (vLLM backend)
  user-data-vllm-snp.yaml     # Cloud-init for local QEMU SNP VMs (vLLM backend)
  user-data-vllm-regular.yaml # Cloud-init for local QEMU regular VMs (vLLM backend)
  debs/                        # Custom kernel .deb packages (SNP with Coconut SVSM only, added manually)
  cloud/                       # Cloud provider configs (GCP, Azure) — see cloud/README.md
```

### Backend Selection

Each CVM mode has two cloud-init variants:
- **Ollama** (default) — `user-data-{mode}.yaml` — lightweight, supports multiple models, good for development
- **vLLM** — `user-data-vllm-{mode}.yaml` — high-performance inference, OpenAI-compatible API, uses GPU if available

The cube agent is backend-agnostic — it proxies requests to whatever `UV_CUBE_AGENT_TARGET_URL` is set to (Ollama on port 11434, vLLM on port 8000).

## Local Deployment (QEMU)

All files in the root of this directory are for **local QEMU-based deployment**. Use `qemu.sh` to launch VMs directly on a host machine with KVM.

### Quick Start

```bash
# Auto-detect CVM support (TDX or SNP) and launch
sudo ./qemu.sh start

# Or force a specific mode
sudo ./qemu.sh start_tdx       # Intel TDX
sudo ./qemu.sh start_regular   # Regular KVM (no CVM)
```

For AMD SEV-SNP, a two-step process is required:

```bash
sudo ./qemu.sh prepare_snp   # Step 1: Install custom kernel via cloud-init (regular KVM)
sudo ./qemu.sh start_snp     # Step 2: Boot the prepared image with IGVM/SNP
```

### Environment Variables

```bash
ENABLE_CVM=tdx sudo ./qemu.sh start       # Force CVM mode: auto, tdx, snp, none
RAM=32768M CPU=16 sudo ./qemu.sh start     # Customize VM resources
IGVM=/path/to/coconut-qemu.igvm sudo ./qemu.sh start_snp  # Override IGVM path
```

### Detect Available CVM Support

```bash
sudo ./qemu.sh detect
```

### After First Boot

Default SSH access:
- **Port**: 6190 (forwarded from guest port 22)
- **User**: ultraviolet
- **Password**: password

## Cloud Deployment (GCP / Azure)

For deploying on cloud providers, see [cloud/README.md](cloud/README.md). Cloud providers handle confidential computing at the hypervisor level, so no custom kernel, IGVM, or module loading is needed.

## CVM Details

### Intel TDX

- Ubuntu 24.04 kernel has `CONFIG_INTEL_TDX_GUEST=y` enabled by default — no custom kernel needed
- **Local QEMU**: Uses `user-data-tdx.yaml` with `qemu.sh start_tdx`; loads `tdx_guest` module via modprobe
- **Cloud**: TDX module loading is handled by the cloud provider

### AMD SEV-SNP

- **Local QEMU with Coconut SVSM**: Requires a custom kernel and two-step boot process (see [SNP Custom Kernel](#snp-custom-kernel) below). Uses `user-data-snp.yaml` which loads `sev-guest` and `ccp` modules via modprobe
- **Cloud (GCP/Azure)**: SEV-SNP is enabled at the hypervisor level — no custom kernel, IGVM, or module loading needed. Use configs in `cloud/`

#### SNP Custom Kernel

A custom kernel is only required when the **local** SNP host runs Coconut SVSM. The standard Ubuntu 24.04 kernel does not support Coconut SVSM.

A custom-built kernel is required with the following configuration options enabled:

- `CONFIG_AMD_MEM_ENCRYPT=y` — AMD memory encryption support
- `CONFIG_SEV_GUEST=y` — SEV guest driver
- `CONFIG_TCG_PLATFORM=y` — vTPM support
- Coconut SVSM guest support patches applied

The kernel must be packaged as `.deb` files (`linux-image-*.deb`, `linux-headers-*.deb`).

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

The `prepare_snp` command automatically packages the debs along with `user-data` and `meta-data` into the seed ISO using `genisoimage`:

```bash
genisoimage -output seed.img -volid cidata -rock <cidata-dir>/
```

On first boot (`prepare_snp`), cloud-init mounts the seed ISO, installs the `.deb` packages, and runs `update-grub`. Then `start_snp` boots the prepared image with IGVM/SNP.

**Dependency:**

```bash
sudo apt-get install genisoimage
```

## Host Requirements (Local QEMU Only)

These requirements apply only to local QEMU deployment. Cloud providers manage these at the infrastructure level.

### For TDX VMs
- Intel CPU with TDX support (4th Gen Xeon Scalable or newer)
- TDX-enabled BIOS/firmware
- Host kernel with TDX module initialized

### For SNP VMs
- AMD EPYC CPU with SEV-SNP support (Milan or newer)
- SEV-SNP enabled in BIOS
- Host kernel with SEV-SNP/SVSM support
- `/dev/sev` device available
- Coconut SVSM QEMU binary at `<path-to-svsm-qemu-dir>/qemu-svsm/bin/qemu-system-x86_64`
- IGVM file at `/etc/cocos/coconut-qemu.igvm` (or set `IGVM` env var)
- `genisoimage` installed (`apt-get install genisoimage`)
- Custom kernel `.deb` files in `debs/` (see [SNP Custom Kernel](#snp-custom-kernel))

### Common
- QEMU with confidential computing support
- OVMF firmware (for UEFI boot)
- KVM enabled
