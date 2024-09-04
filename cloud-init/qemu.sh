#!/bin/bash

BASE_IMAGE_URL="https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
BASE_IMAGE="ubuntu-base.qcow2"
CUSTOM_IMAGE="ubuntu-custom.qcow2"
DISK_SIZE="35G"
SEED_IMAGE="seed.img"
USER_DATA="user-data"
META_DATA="meta-data"
VM_NAME="vault-ai-vm"
RAM="24576"
CPU="6"
USER="ultraviolet"
PASSWORD="password"

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
  - cd /home/$USER
  - wget https://gist.rodneyosodo.com/rodneyosodo/78f602893ccb493ba27bfd9d180adfbb/raw/HEAD/ollama-compose.yaml -O ollama-compose.yaml
  - docker compose -f ollama-compose.yaml pull

final_message: "Docker installation complete."
EOF

cat <<EOF > $META_DATA
instance-id: iid-${VM_NAME}
local-hostname: $VM_NAME
EOF

echo "Creating cloud-init seed image..."
cloud-localds $SEED_IMAGE $USER_DATA $META_DATA

echo "Starting QEMU VM..."
qemu-system-x86_64 \
  -name $VM_NAME \
  -m $RAM \
  -smp $CPU \
  -cpu host \
  -machine q35 \
  -enable-kvm \
  -drive file=${CUSTOM_IMAGE},if=virtio,format=qcow2 \
  -cdrom $SEED_IMAGE \
  -boot d \
  -netdev user,id=net0,hostfwd=tcp::6190-:22,hostfwd=tcp::6191-:11434,hostfwd=tcp::6192-:3000 \
  -device e1000,netdev=net0 \
  -vnc :9 \
  -nographic \
  -no-reboot 
