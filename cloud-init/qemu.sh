#!/bin/bash

BASE_IMAGE_URL="https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
BASE_IMAGE="ubuntu-base.qcow2"
CUSTOM_IMAGE="ubuntu-custom.qcow2"
DISK_SIZE="35G"
SEED_IMAGE="seed.img"
USER_DATA="user-data"
META_DATA="meta-data"
VM_NAME="vault-ai-vm"
RAM="24576M"
CPU="6"
USER="ultraviolet"
PASSWORD="password"
QEMU_BIN="/home/cocosai/danko/AMDSEV/usr/local/bin/qemu-system-x86_64"
OVMF_CODE="/usr/share/OVMF/OVMF_CODE.fd"
OVMF_VARS="/usr/share/OVMF/OVMF_VARS.fd"

if ! command -v wget &> /dev/null; then
  echo "wget is not installed. Please install it and try again."
  exit 1
fi

if ! command -v cloud-localds &> /dev/null; then
  echo "cloud-localds is not installed. Please install it and try again."
  exit 1
fi

if ! command -v qemu-system-x86_64 &> /dev/null; then
  echo "qemu-system-x86_64 is not installed. Please install it and try again."
  exit 1
fi

if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root" 1>&2
   exit 1
fi

if [ ! -f $QEMU_BIN ]; then
  echo "QEMU binary not found. Please install it and try again."
  exit 1
fi

if [ ! -f $OVMF_CODE ]; then
  echo "OVMF code not found. Please install it and try again."
  exit 1
fi

if [ ! -f $OVMF_VARS ]; then
  echo "OVMF vars not found. Please install it and try again."
  exit 1
fi

if [ ! -f $BASE_IMAGE ]; then
  echo "Downloading base Ubuntu image..."
  wget -q $BASE_IMAGE_URL -O $BASE_IMAGE
fi

echo "Creating custom QEMU image..."
qemu-img create -f qcow2 -b $BASE_IMAGE -F qcow2 $CUSTOM_IMAGE $DISK_SIZE

cat <<EOF > $USER_DATA
#cloud-config
package_update: true
package_upgrade: true

users:
  - default
  - name: $USER
    gecos: Default User
    groups:
      - sudo
    sudo:
      - ALL=(ALL:ALL) ALL
    shell: /bin/bash

chpasswd:
  list: |
    $USER:$PASSWORD
  expire: False

ssh_pwauth: True

packages:
  - curl

runcmd:
  - curl -fsSL https://get.docker.com -o get-docker.sh
  - sh ./get-docker.sh
  - groupadd docker
  - usermod -aG docker $USER
  - newgrp docker

final_message: "Docker installation complete."
EOF

cat <<EOF > $META_DATA
instance-id: iid-${VM_NAME}
local-hostname: $VM_NAME
EOF

echo "Creating cloud-init seed image..."
cloud-localds $SEED_IMAGE $USER_DATA $META_DATA

echo "Starting QEMU VM..."
$QEMU_BIN \
  -name $VM_NAME \
  -m $RAM,slots=5,maxmem=30G \
  -smp $CPU,maxcpus=24 \
  -cpu EPYC \
  -machine q35 \
  -enable-kvm \
  -boot d \
  -netdev user,id=vmnic,hostfwd=tcp::6190-:22,hostfwd=tcp::6191-:11434,hostfwd=tcp::6192-:3000 \
  -device e1000,netdev=vmnic,romfile= \
  -vnc :9 \
  -nographic \
  -no-reboot \
  -drive if=pflash,format=raw,unit=0,file="$OVMF_CODE",readonly=on \
  -drive if=pflash,format=raw,unit=1,file="$OVMF_VARS" \
  -drive file=$SEED_IMAGE,media=cdrom \
  -drive file=$CUSTOM_IMAGE,if=none,id=disk0,format=qcow2 \
  -device virtio-scsi-pci,id=scsi,disable-legacy=on,iommu_platform=true \
  -device scsi-hd,drive=disk0 \
  -object sev-guest,id=sev0,cbitpos=51,reduced-phys-bits=1,discard=none,kernel-hashes=off \
  -machine memory-encryption=sev0
