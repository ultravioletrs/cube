#!/bin/bash
# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0
#
# QEMU launch script for Ubuntu cloud images with CVM (TDX/SNP) support

set -e

# Default configuration
BASE_IMAGE_URL="https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
BASE_IMAGE="ubuntu-base.qcow2"
CUSTOM_IMAGE="ubuntu-custom.qcow2"
DISK_SIZE="35G"
SEED_IMAGE="seed.img"
META_DATA="meta-data"
VM_NAME="cube-ai-vm"
RAM="16384M"
CPU="8"
USER="ultraviolet"
PASSWORD="password"
QEMU_BINARY="qemu-system-x86_64"
OVMF_CODE="/usr/share/OVMF/OVMF_CODE.fd"
OVMF_VARS="/usr/share/OVMF/OVMF_VARS.fd"
OVMF_VARS_COPY="OVMF_VARS.fd"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# CVM mode: auto, tdx, snp, none
ENABLE_CVM="${ENABLE_CVM:-auto}"

function check_dependencies() {
    local missing=()

    if ! command -v wget &> /dev/null; then
        missing+=("wget")
    fi

    if ! command -v cloud-localds &> /dev/null; then
        missing+=("cloud-localds (cloud-image-utils)")
    fi

    if ! command -v qemu-system-x86_64 &> /dev/null; then
        missing+=("qemu-system-x86_64")
    fi

    if [ ${#missing[@]} -ne 0 ]; then
        echo "Missing dependencies: ${missing[*]}"
        echo "Please install them and try again."
        exit 1
    fi
}

function check_root() {
    if [[ $EUID -ne 0 ]]; then
        echo "This script must be run as root" 1>&2
        exit 1
    fi
}

function detect_cvm_support() {
    local tdx_available=false
    local snp_available=false

    # Check for TDX host support
    if dmesg 2>/dev/null | grep -q "virt/tdx: module initialized"; then
        tdx_available=true
        echo "TDX host support detected (module initialized)"
    elif grep -q tdx /proc/cpuinfo 2>/dev/null; then
        tdx_available=true
        echo "TDX CPU support detected"
    fi

    # Check for SEV-SNP host support
    if [ -e /dev/sev ]; then
        snp_available=true
        echo "SEV device detected"
    fi
    if dmesg 2>/dev/null | grep -q "SEV-SNP supported"; then
        snp_available=true
        echo "SEV-SNP host support detected"
    elif grep -q sev /proc/cpuinfo 2>/dev/null; then
        snp_available=true
        echo "SEV CPU support detected"
    fi

    # Return detected support
    if [ "$tdx_available" = true ]; then
        echo "tdx"
    elif [ "$snp_available" = true ]; then
        echo "snp"
    else
        echo "none"
    fi
}

function download_base_image() {
    if [ ! -f "$BASE_IMAGE" ]; then
        echo "Downloading base Ubuntu image..."
        wget -q --show-progress "$BASE_IMAGE_URL" -O "$BASE_IMAGE"
    else
        echo "Base image already exists: $BASE_IMAGE"
    fi
}

function create_custom_image() {
    echo "Creating custom QEMU image..."
    qemu-img create -f qcow2 -b "$BASE_IMAGE" -F qcow2 "$CUSTOM_IMAGE" "$DISK_SIZE"
}

function create_ovmf_vars_copy() {
    if [ ! -f "$OVMF_VARS_COPY" ]; then
        echo "Creating OVMF vars copy..."
        cp "$OVMF_VARS" "$OVMF_VARS_COPY"
    fi
}

function create_seed_image() {
    local user_data_file="$1"

    echo "Creating seed image with $user_data_file..."

    # Create meta-data
    cat <<EOF > "$META_DATA"
instance-id: iid-${VM_NAME}
local-hostname: $VM_NAME
EOF

    cloud-localds "$SEED_IMAGE" "$user_data_file" "$META_DATA"
}

function create_seed_image_snp() {
    local user_data_file="$1"

    echo "Creating SNP seed image with $user_data_file..."

    if ! command -v genisoimage &> /dev/null; then
        echo "genisoimage is not installed. Please install genisoimage and try again."
        exit 1
    fi

    local cidata_dir
    cidata_dir="$(mktemp -d)"
    trap "rm -rf $cidata_dir" RETURN

    # Write meta-data
    cat <<EOF > "$cidata_dir/meta-data"
instance-id: iid-${VM_NAME}
local-hostname: $VM_NAME
EOF

    # Copy user-data
    cp "$user_data_file" "$cidata_dir/user-data"

    # Include kernel .deb files if present (required for SNP-compatible kernel)
    if [ -d "${SCRIPT_DIR}/debs" ] && ls "${SCRIPT_DIR}/debs"/*.deb 2>/dev/null; then
        mkdir -p "$cidata_dir/debs"
        cp "${SCRIPT_DIR}/debs"/*.deb "$cidata_dir/debs/"
        echo "Included kernel .deb files from ${SCRIPT_DIR}/debs/"
    fi

    genisoimage -output "$SEED_IMAGE" -volid cidata -rock "$cidata_dir"
}

function start_regular() {
    echo "Starting QEMU VM in regular mode (no CVM)..."

    create_ovmf_vars_copy
    create_seed_image "${SCRIPT_DIR}/user-data-regular.yaml"

    $QEMU_BINARY \
        -name "$VM_NAME" \
        -m "$RAM" \
        -smp "$CPU" \
        -enable-kvm \
        -boot d \
        -cpu host \
        -machine q35 \
        -drive if=pflash,format=raw,unit=0,file="$OVMF_CODE",readonly=on \
        -drive if=pflash,format=raw,unit=1,file="$OVMF_VARS_COPY" \
        -netdev user,id=vmnic,hostfwd=tcp::6190-:22,hostfwd=tcp::6191-:80,hostfwd=tcp::6192-:443,hostfwd=tcp::6193-:7001 \
        -device virtio-net-pci,netdev=vmnic,romfile= \
        -nographic \
        -no-reboot \
        -drive file="$SEED_IMAGE",media=cdrom \
        -drive file="$CUSTOM_IMAGE",if=none,id=disk0,format=qcow2 \
        -device virtio-scsi-pci,id=scsi,disable-legacy=on \
        -device scsi-hd,drive=disk0
}

function start_tdx() {
    echo "Starting QEMU VM with Intel TDX (Confidential VM)..."

    create_seed_image "${SCRIPT_DIR}/user-data-tdx.yaml"

    $QEMU_BINARY \
        -name "$VM_NAME,process=$VM_NAME,debug-threads=on" \
        -m "$RAM" \
        -smp "$CPU" \
        -enable-kvm \
        -cpu host,pmu=off \
        -object memory-backend-memfd,id=ram1,size="$RAM",share=true,prealloc=false \
        -object '{"qom-type":"tdx-guest","id":"tdx0","quote-generation-socket":{"type":"vsock","cid":"2","port":"4050"}}' \
        -machine q35,confidential-guest-support=tdx0,memory-backend=ram1,kernel-irqchip=split,hpet=off \
        -bios /usr/share/ovmf/OVMF.fd \
        -netdev user,id=vmnic,hostfwd=tcp::6190-:22,hostfwd=tcp::6191-:80,hostfwd=tcp::6192-:443,hostfwd=tcp::6193-:7001 \
        -device virtio-net-pci,disable-legacy=on,iommu_platform=true,netdev=vmnic,romfile= \
        -nodefaults \
        -nographic \
        -serial mon:stdio \
        -monitor pty \
        -no-reboot \
        -drive file="$SEED_IMAGE",media=cdrom \
        -drive file="$CUSTOM_IMAGE",if=none,id=disk0,format=qcow2 \
        -device virtio-scsi-pci,id=scsi,disable-legacy=on,iommu_platform=true \
        -device scsi-hd,drive=disk0 \
        -device vhost-vsock-pci,guest-cid=3
}

function prepare_snp() {
    echo "Preparing SNP image via cloud-init (regular KVM, no SNP)..."
    echo "This installs the custom kernel into the image. Run start_snp after this completes."

    local SNP_QEMU="<path-to-svsm-qemu-dir>/qemu-svsm/bin/qemu-system-x86_64"

    create_seed_image_snp "${SCRIPT_DIR}/user-data-snp.yaml"

    $SNP_QEMU \
        -name "$VM_NAME" \
        -enable-kvm \
        -m 2048 \
        -smp 2 \
        -drive "file=$CUSTOM_IMAGE,if=virtio,format=qcow2" \
        -drive "file=$SEED_IMAGE,if=virtio,format=raw,readonly=on" \
        -netdev "user,id=n0,hostfwd=tcp::6190-:22" \
        -device virtio-net-pci,netdev=n0 \
        -nographic
}

function start_snp() {
    echo "Starting QEMU VM with AMD SEV-SNP (Confidential VM)..."
    echo "Ensure prepare_snp has been run first to install the custom kernel."

    local SNP_QEMU="<path-to-svsm-qemu-dir>/qemu-svsm/bin/qemu-system-x86_64"
    local IGVM="${IGVM:-/etc/cocos/coconut-qemu.igvm}"

    $SNP_QEMU \
        -name "$VM_NAME" \
        -enable-kvm \
        -cpu EPYC-v4 \
        -machine q35 \
        -smp "$CPU,sockets=1,threads=1" \
        -m "$RAM,slots=5,maxmem=40G" \
        -netdev user,id=vmnic,hostfwd=tcp::6190-:22,hostfwd=tcp::6191-:80,hostfwd=tcp::6192-:443,hostfwd=tcp::6193-:7001 \
        -device virtio-net-pci,disable-legacy=on,iommu_platform=true,netdev=vmnic,romfile= \
        -machine confidential-guest-support=sev0,memory-backend=ram1,igvm-cfg=igvm0 \
        -object memory-backend-memfd,id=ram1,size="$RAM",share=true,prealloc=false,reserve=false \
        -object sev-snp-guest,id=sev0,cbitpos=51,reduced-phys-bits=1 \
        -object igvm-cfg,id=igvm0,file="$IGVM" \
        -drive file="$CUSTOM_IMAGE",if=none,id=disk0,format=qcow2 \
        -device virtio-scsi-pci,id=scsi0,disable-legacy=on,iommu_platform=true \
        -device scsi-hd,drive=disk0,bus=scsi0.0 \
        -device vhost-vsock-pci,id=vhost-vsock-pci0,guest-cid=198 \
        -nographic \
        -no-reboot \
        -monitor pty
}

function print_help() {
    cat <<EOF
Usage: $0 [command] [options]

Commands:
  start         Start the QEMU VM (auto-detect CVM support)
  start_tdx     Start the QEMU VM with Intel TDX enabled
  prepare_snp   Prepare the image for SNP (run once before start_snp)
  start_snp     Start the QEMU VM with AMD SEV-SNP enabled (requires prepare_snp)
  start_regular Start the QEMU VM without CVM (regular KVM)
  detect        Detect available CVM support on this host
  help          Show this help message

Environment Variables:
  ENABLE_CVM    Force CVM mode: auto (default), tdx, snp, none
  RAM           VM RAM size (default: 16384M)
  CPU           Number of vCPUs (default: 8)
  DISK_SIZE     Disk size (default: 35G)

Backend:
  Default cloud-init configs use Ollama. For vLLM, use the
  user-data-vllm-*.yaml variants (uses GPU if available).

Examples:
  $0 start              # Auto-detect and start with best available CVM
  $0 start_tdx          # Force TDX mode
  $0 start_snp          # Force SNP mode
  ENABLE_CVM=none $0 start  # Disable CVM, use regular KVM
EOF
}

function main() {
    check_dependencies
    check_root
    download_base_image
    create_custom_image

    local cmd="${1:-help}"

    case "$cmd" in
        start)
            local detected
            if [ "$ENABLE_CVM" = "auto" ]; then
                detected=$(detect_cvm_support)
            else
                detected="$ENABLE_CVM"
            fi

            case "$detected" in
                tdx)
                    echo "CVM mode: TDX"
                    start_tdx
                    ;;
                snp)
                    echo "CVM mode: SNP"
                    start_snp
                    ;;
                none|*)
                    echo "CVM mode: None (regular KVM)"
                    start_regular
                    ;;
            esac
            ;;
        start_tdx)
            start_tdx
            ;;
        prepare_snp)
            prepare_snp
            ;;
        start_snp)
            start_snp
            ;;
        start_regular)
            start_regular
            ;;
        detect)
            echo "Detecting CVM support..."
            detected=$(detect_cvm_support)
            echo "Detected: $detected"
            ;;
        help|--help|-h)
            print_help
            ;;
        *)
            echo "Unknown command: $cmd"
            print_help
            exit 1
            ;;
    esac
}

main "$@"
