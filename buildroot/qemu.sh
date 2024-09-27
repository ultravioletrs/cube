#!/bin/bash

VM_NAME="cube-ai-vm"
RAM="10240M"
CPU="4"

if ! command -v qemu-system-x86_64 &> /dev/null; then
  echo "qemu-system-x86_64 is not installed. Please install it and try again."
  exit 1
fi

echo "Starting QEMU VM..."
qemu-system-x86_64 \
  -name $VM_NAME \
  -m $RAM \
  -smp $CPU \
  -cpu EPYC \
  -machine q35 \
  -enable-kvm \
  -boot d \
  -netdev user,id=vmnic,hostfwd=tcp::6190-:22,hostfwd=tcp::6191-:80,hostfwd=tcp::6192-:443,hostfwd=tcp::6193-:3001 \
  -device e1000,netdev=vmnic,romfile= \
  -vnc :9 \
  -nographic \
  -no-reboot \
  -drive file=$SEED_IMAGE,media=cdrom \
  -drive file=$CUSTOM_IMAGE,if=none,id=disk0,format=qcow2 \
  -device virtio-scsi-pci,id=scsi,disable-legacy=on,iommu_platform=true \
  -device scsi-hd,drive=disk0 \
  -machine memory-encryption=sev0,confidential-guest-support=sev0 \
  -object sev-guest,id=sev0,cbitpos=51,reduced-phys-bits=1 \
  -drive if=pflash,format=raw,unit=0,file=/usr/share/OVMF/OVMF_CODE.fd,readonly=on \
  -drive if=pflash,format=raw,unit=1,file=/usr/share/OVMF/OVMF_VARS.fd


/home/cocosai/danko/AMDSEV/usr/local/bin/qemu-system-x86_64 \
  -name cube-ai-vm \
  -m 10240M \
  -smp 8 \
  -cpu EPYC-v4 \
  -machine q35 \
  -enable-kvm \
  -boot d \
  -netdev user,id=vmnic,hostfwd=tcp::6190-:22,hostfwd=tcp::6191-:80,hostfwd=tcp::6192-:443,hostfwd=tcp::6193-:3001 \
  -device e1000,netdev=vmnic,romfile= \
  -vnc :9 \
  -nographic \
  -no-reboot \
  -drive file=ubuntu-custom.qcow2,if=none,id=disk0,format=qcow2 \
  -device virtio-scsi-pci,id=scsi,disable-legacy=on,iommu_platform=true \
  -device scsi-hd,drive=disk0 \
  -machine memory-encryption=sev0,vmport=off \
  -object memory-backend-memfd-private,id=ram1,size=10240M,share=true \
  -object sev-snp-guest,id=sev0,cbitpos=51,reduced-phys-bits=1 \
  -machine memory-backend=ram1,kvm-type=protected \
  -drive if=pflash,format=raw,unit=0,file=./OVMF_CODE.fd,readonly=on \
  -drive if=pflash,format=raw,unit=1,file=./OVMF_VARS.fd

/home/cocosai/danko/AMDSEV/usr/local/bin/qemu-system-x86_64 \
  -enable-kvm \
  -machine q35 \
  -cpu EPYC-v4 \
  -smp 4,maxcpus=16 \
  -m 25G,slots=5,maxmem=30G \
  -drive if=pflash,format=raw,unit=0,file=/home/cocosai/danko/AMDSEV/ovmf/Build/AmdSev/DEBUG_GCC5/FV/OVMF.fd,readonly=on \
  -netdev user,id=vmnic-ed3cd402-d78e-4136-8070-96c03affc0aa,hostfwd=tcp::6100-:7002 \
  -device virtio-net-pci,disable-legacy=on,iommu_platform=true,netdev=vmnic-ed3cd402-d78e-4136-8070-96c03affc0aa,addr=0x2,romfile= \
  -device vhost-vsock-pci,id=vhost-vsock-pci0,guest-cid=3 \
  -object memory-backend-memfd-private,id=ram1,size=25G,share=true \
  -machine memory-backend=ram1,kvm-type=protected \
  -kernel /home/sammy/bzImage \
  -append "quiet console=null rootfstype=ramfs" \
  -initrd /home/sammy/rootfs.cpio.gz \
  -object sev-snp-guest,id=sev0-ed3cd402-d78e-4136-8070-96c03affc0aa,cbitpos=51,reduced-phys-bits=1,discard=none,kernel-hashes=on,host-data=FTZtWfgKU2WimWFajBIdIUtKTcxy5xCMBNxex6sFf/4= \
  -machine memory-encryption=sev0-ed3cd402-d78e-4136-8070-96c03affc0aa \
  -nographic \
  -monitor pty
