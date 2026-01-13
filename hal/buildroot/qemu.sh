#!/bin/bash
# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

VM_NAME="cube-ai-vm"
RAM="10240M"
CPU="4"
CPU_TYPE="EPYC-v4"
QEMU_AMDSEV_BINARY="/usr/bin/qemu-system-x86_64"
QEMU_OVMF_CODE="/var/cube-ai/OVMF.fd"
KERNEL_PATH="/etc/cube/bzImage"
# INITRD_PATH="../../buildroot/output/images/rootfs.cpio.gz" # Unused for disk boot
FS_PATH="/etc/cube/rootfs.ext4"
QEMU_APPEND_ARG="root=/dev/vda rw console=ttyS0"

function check(){
    if [ ! -f "$FS_PATH" ]; then
    echo "rootfs.ext4 file not found. Please build it (make) and try again."
        exit 1
    fi

    if [ ! -f "$KERNEL_PATH" ]; then
        echo "bzImage file not found. Please build it and try again."
        exit 1
    fi
}

function start_qemu(){
    if ! command -v qemu-system-x86_64 &> /dev/null; then
        echo "qemu-system-x86_64 is not installed. Please install it and try again."
        exit 1
    fi

    check

    echo "Starting QEMU VM (Disk Boot)..."

    qemu-system-x86_64 \
    -name $VM_NAME \
    -m $RAM \
    -smp $CPU \
    -cpu $CPU_TYPE \
    -machine q35 \
    -enable-kvm \
    -netdev user,id=vmnic,hostfwd=tcp::6190-:22,hostfwd=tcp::6191-:80,hostfwd=tcp::6192-:443,hostfwd=tcp::6193-:7001,hostfwd=tcp::6194-:11434,hostfwd=tcp::6195-:8000,dns=8.8.8.8 \
    -device virtio-net-pci,disable-legacy=on,iommu_platform=true,netdev=vmnic,romfile= \
    -nographic \
    -no-reboot \
    -kernel $KERNEL_PATH \
    -drive file=$FS_PATH,format=raw,if=virtio,index=0  \
    -append "$QEMU_APPEND_ARG"
}

function start_cvm(){
    if ! command -v $QEMU_AMDSEV_BINARY &> /dev/null; then
        echo "QEMU binary not found at $QEMU_AMDSEV_BINARY"
        exit 1
    fi

    check

    echo "Starting QEMU CVM (SEV-SNP Disk Boot)..."

    $QEMU_AMDSEV_BINARY \
    -name $VM_NAME \
    -m $RAM \
    -smp $CPU \
    -cpu $CPU_TYPE \
    -machine q35 \
    -enable-kvm \
    -netdev user,id=vmnic,hostfwd=tcp::6190-:22,hostfwd=tcp::6191-:80,hostfwd=tcp::6192-:443,hostfwd=tcp::6193-:7001,hostfwd=tcp::6194-:11434,hostfwd=tcp::6195-:8000,dns=8.8.8.8 \
    -device virtio-net-pci,disable-legacy=on,iommu_platform=true,netdev=vmnic,romfile= \
    -nographic \
    -no-reboot \
    -kernel $KERNEL_PATH \
    -drive file=$FS_PATH,format=raw,if=virtio \
    -drive if=pflash,format=raw,unit=0,file=$QEMU_OVMF_CODE,readonly=on \
    -device vhost-vsock-pci,id=vhost-vsock-pci0,guest-cid=198 \
    -object memory-backend-memfd-private,id=ram1,size=$RAM,share=true \
    -machine memory-encryption=sev0 \
    -machine memory-backend=ram1,kvm-type=protected \
    -object sev-snp-guest,id=sev0,cbitpos=51,reduced-phys-bits=1,discard=none,kernel-hashes=on \
    -append "$QEMU_APPEND_ARG"
}

function start_tdx(){ 
    $QEMU_AMDSEV_BINARY \
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
    -netdev user,id=nic0_td,hostfwd=tcp::6190-:22,hostfwd=tcp::6191-:80,hostfwd=tcp::6192-:443,hostfwd=tcp::6193-:7001,hostfwd=tcp::6194-:11434,hostfwd=tcp::6195-:8000,dns=8.8.8.8 \
    -kernel $KERNEL_PATH \
    -append "$QEMU_APPEND_ARG" \
    -object memory-backend-memfd,id=mem0,size=20G \
    -drive file=$FS_PATH,format=raw,if=virtio \
    -device vhost-vsock-pci,guest-cid=6 \
    -monitor pty \
    -monitor unix:monitor,server,nowait
}

function generate_snp_expected_measurement(){
    if ! command -v sev-snp-measure &> /dev/null; then
        echo "sev-snp-measure is not installed. Please install it and try again."
        exit 1
    fi

    echo "Generating expected measurement..."
    sev-snp-measure \
    --mode snp \
    --vcpus=$CPU \
    --vcpu-type=$CPU_TYPE \
    --ovmf=$QEMU_OVMF_CODE \
    --kernel=$KERNEL_PATH \
    --append="$QEMU_APPEND_ARG"
}

function print_help(){
    echo "Usage: $0 [command]"
    echo "Commands:"
    echo "  start: Start the QEMU VM"
    echo "  start_cvm: Start the QEMU VM with AMD SEV-SNP enabled"
    echo "  measure: Use sev-snp-measure utility to calculate the expected measurement"
    echo "  check: Check if the required files are present"
}

if [ $# -eq 0 ]; then
    print_help
    exit 0
fi

if [ $# -gt 0 ]; then
    case "$1" in
        "start")
            start_qemu
            ;;
        "check")
            check
            ;;
        "start_cvm")
            start_cvm
            ;;
        "start_tdx")
            start_tdx
            ;;
        "measure")
            generate_snp_expected_measurement
            ;;
        *)
            echo "Unknown command: $1"
            exit 1
            ;;
    esac
fi
