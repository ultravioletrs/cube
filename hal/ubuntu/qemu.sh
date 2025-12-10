#!/bin/bash
# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

BASE_IMAGE_URL="https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
BASE_IMAGE="ubuntu-base.qcow2"
CUSTOM_IMAGE="ubuntu-custom.qcow2"
DISK_SIZE="35G"
SEED_IMAGE="seed.img"
USER_DATA="user-data"
META_DATA="meta-data"
VM_NAME="cube-ai-vm"
RAM="16384M"
CPU="8"
USER="ultraviolet"
PASSWORD="password"
QEMU_BINARY="qemu-system-x86_64"
OVMF_CODE="/usr/share/OVMF/OVMF_CODE.fd"
OVMF_VARS="/usr/share/OVMF/OVMF_VARS.fd"
OVMF_VARS_COPY="OVMF_VARS.fd"  # Per-VM copy of OVMF vars
ENABLE_CVM="${ENABLE_CVM:-auto}"  # Options: auto, tdx, none

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

if [ ! -f $BASE_IMAGE ]; then
  echo "Downloading base Ubuntu image..."
  wget -q $BASE_IMAGE_URL -O $BASE_IMAGE
fi

echo "Creating custom QEMU image..."
qemu-img create -f qcow2 -b $BASE_IMAGE -F qcow2 $CUSTOM_IMAGE $DISK_SIZE

# Create a writable copy of OVMF_VARS for this VM instance
if [ ! -f $OVMF_VARS_COPY ]; then
  echo "Creating OVMF vars copy..."
  cp $OVMF_VARS $OVMF_VARS_COPY
fi

# We don't upgrade the system since this changes initramfs
cat <<'EOF' > $USER_DATA
#cloud-config
package_update: true
package_upgrade: false

users:
  - name: ultraviolet
    plain_text_passwd: password
    lock_passwd: false
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
  - name: ollama
    system: true
    home: /var/lib/ollama
    shell: /usr/sbin/nologin

ssh_pwauth: true

packages:
  - curl
  - git
  - golang-go
  - build-essential

write_files:
  - path: /etc/cube/agent.env
    content: |
      UV_CUBE_AGENT_LOG_LEVEL=info
      UV_CUBE_AGENT_HOST=0.0.0.0
      UV_CUBE_AGENT_PORT=7001
      UV_CUBE_AGENT_INSTANCE_ID=cube-agent-01
      UV_CUBE_AGENT_TARGET_URL=http://localhost:11434
      UV_CUBE_AGENT_SERVER_CERT=/etc/cube/certs/server.crt
      UV_CUBE_AGENT_SERVER_KEY=/etc/cube/certs/server.key
      UV_CUBE_AGENT_SERVER_CA_CERTS=/etc/cube/certs/ca.crt
      UV_CUBE_AGENT_CA_URL=https://prism.ultraviolet.rs/am-certs
    permissions: '0644'
  - path: /etc/systemd/system/ollama.service
    content: |
      [Unit]
      Description=Ollama Service
      After=network-online.target
      Wants=network-online.target

      [Service]
      Type=simple
      User=ollama
      Group=ollama
      Environment="OLLAMA_HOST=0.0.0.0:11434"
      ExecStart=/usr/local/bin/ollama serve
      Restart=always
      RestartSec=3

      [Install]
      WantedBy=multi-user.target
    permissions: '0644'
  - path: /etc/systemd/system/cube-agent.service
    content: |
      [Unit]
      Description=Cube Agent Service
      After=network-online.target ollama.service
      Wants=network-online.target

      [Service]
      Type=simple
      EnvironmentFile=/etc/cube/agent.env
      ExecStart=/usr/local/bin/cube-agent
      Restart=on-failure
      RestartSec=5
      StartLimitBurst=5
      StartLimitIntervalSec=60

      [Install]
      WantedBy=multi-user.target
    permissions: '0644'
  - path: /usr/local/bin/pull-ollama-models.sh
    content: |
      #!/bin/bash
      # Wait for ollama to be ready
      for i in $(seq 1 60); do
        if curl -s http://localhost:11434/api/version > /dev/null 2>&1; then
          break
        fi
        sleep 2
      done
      # Pull models
      /usr/local/bin/ollama pull tinyllama:1.1b
      /usr/local/bin/ollama pull starcoder2:3b
      /usr/local/bin/ollama pull nomic-embed-text:v1.5
    permissions: '0755'

runcmd:
  - echo 'ultraviolet:password' | chpasswd
  - |
    cat > /etc/ssh/sshd_config.d/60-cloudimg-settings.conf << 'SSHEOF'
    PasswordAuthentication yes
    SSHEOF
  - systemctl restart sshd
  - sleep 2
  - |
    # Install TDX-capable kernel from Ubuntu's intel-tdx PPA
    add-apt-repository -y ppa:kobuk-team/intel-tdx || echo "PPA add failed, trying canonical tdx"
    apt-get update || true
    apt-get install -y linux-image-generic linux-modules-extra-generic || echo "Kernel install failed"
    # Try to load TDX guest module
    modprobe tdx_guest 2>/dev/null || echo "tdx_guest module not yet available (may need reboot)"
    # Add to modules to load at boot
    mkdir -p /etc/modules-load.d
    echo "tdx_guest" > /etc/modules-load.d/tdx.conf
  - mkdir -p /etc/cube
  - mkdir -p /etc/cube/certs
  - |
    # Generate CA certificate
    openssl req -x509 -newkey rsa:4096 -keyout /etc/cube/certs/ca.key -out /etc/cube/certs/ca.crt -days 365 -nodes -subj "/CN=Cube-CA"
    # Generate server certificate
    openssl req -newkey rsa:4096 -keyout /etc/cube/certs/server.key -out /etc/cube/certs/server.csr -nodes -subj "/CN=cube-agent"
    openssl x509 -req -in /etc/cube/certs/server.csr -CA /etc/cube/certs/ca.crt -CAkey /etc/cube/certs/ca.key -CAcreateserial -out /etc/cube/certs/server.crt -days 365
    # Generate client certificate for mTLS
    openssl req -newkey rsa:4096 -keyout /etc/cube/certs/client.key -out /etc/cube/certs/client.csr -nodes -subj "/CN=cube-client"
    openssl x509 -req -in /etc/cube/certs/client.csr -CA /etc/cube/certs/ca.crt -CAkey /etc/cube/certs/ca.key -CAcreateserial -out /etc/cube/certs/client.crt -days 365
    # Set permissions
    chmod 600 /etc/cube/certs/*.key
    chmod 644 /etc/cube/certs/*.crt
  - mkdir -p /var/lib/ollama
  - mkdir -p /home/ollama/.ollama
  - chown -R ollama:ollama /var/lib/ollama
  - chown -R ollama:ollama /home/ollama
  - curl -fsSL https://ollama.com/install.sh | sh
  - git clone https://github.com/ultravioletrs/cube.git /tmp/cube
  - cd /tmp/cube && git fetch origin pull/88/head:pr-88 && git checkout pr-88
  - export HOME=/root
  - cd /tmp/cube && /usr/bin/go build -ldflags="-s -w" -o /usr/local/bin/cube-agent ./cmd/agent
  - systemctl daemon-reload
  - systemctl enable ollama.service
  - systemctl start ollama.service
  - systemctl enable cube-agent.service
  - systemctl start cube-agent.service
  - nohup /usr/local/bin/pull-ollama-models.sh > /var/log/ollama-pull.log 2>&1 &

final_message: "Cube Agent and Ollama services started."
EOF

cat <<EOF > $META_DATA
instance-id: iid-${VM_NAME}
local-hostname: $VM_NAME
EOF

echo "Creating ubuntu seed image..."
cloud-localds $SEED_IMAGE $USER_DATA $META_DATA

# Detect TDX support
TDX_AVAILABLE=false
if [ "$ENABLE_CVM" = "auto" ] || [ "$ENABLE_CVM" = "tdx" ]; then
  # Check if TDX is initialized on the host (for creating guest VMs)
  if dmesg | grep -q "virt/tdx: module initialized"; then
    TDX_AVAILABLE=true
    echo "TDX host support detected"
  elif grep -q tdx /proc/cpuinfo; then
    TDX_AVAILABLE=true
    echo "TDX CPU support detected"
  else
    echo "TDX not available on host"
  fi
fi

# Override if explicitly set
if [ "$ENABLE_CVM" = "tdx" ]; then
  TDX_AVAILABLE=true
  echo "TDX mode forced via ENABLE_CVM=tdx"
elif [ "$ENABLE_CVM" = "none" ]; then
  TDX_AVAILABLE=false
  echo "CVM disabled via ENABLE_CVM=none"
fi

# Build QEMU command based on TDX availability
QEMU_CMD="$QEMU_BINARY"
QEMU_OPTS="-name $VM_NAME"
QEMU_OPTS="$QEMU_OPTS -m $RAM"
QEMU_OPTS="$QEMU_OPTS -smp $CPU"
QEMU_OPTS="$QEMU_OPTS -enable-kvm"
QEMU_OPTS="$QEMU_OPTS -boot d"
QEMU_OPTS="$QEMU_OPTS -netdev user,id=vmnic,hostfwd=tcp::6190-:22,hostfwd=tcp::6191-:80,hostfwd=tcp::6192-:443,hostfwd=tcp::6193-:7001"
QEMU_OPTS="$QEMU_OPTS -nographic"
QEMU_OPTS="$QEMU_OPTS -no-reboot"
QEMU_OPTS="$QEMU_OPTS -drive file=$SEED_IMAGE,media=cdrom"
QEMU_OPTS="$QEMU_OPTS -drive file=$CUSTOM_IMAGE,if=none,id=disk0,format=qcow2"
QEMU_OPTS="$QEMU_OPTS -device virtio-scsi-pci,id=scsi,disable-legacy=on"
QEMU_OPTS="$QEMU_OPTS -device scsi-hd,drive=disk0"

if [ "$TDX_AVAILABLE" = true ]; then
  echo "Starting QEMU VM with Intel TDX (Confidential VM)..."
  # Update the -name option to include process and debug-threads
  QEMU_OPTS=$(echo "$QEMU_OPTS" | sed "s/-name $VM_NAME/-name $VM_NAME,process=$VM_NAME,debug-threads=on/")
  # Remove -m and add memory-backend-memfd for TDX (critical!)
  QEMU_OPTS=$(echo "$QEMU_OPTS" | sed "s/-m $RAM//")
  QEMU_OPTS="$QEMU_OPTS -object memory-backend-memfd,id=ram1,size=$RAM,share=true,prealloc=false"
  QEMU_OPTS="$QEMU_OPTS -m $RAM"
  QEMU_OPTS="$QEMU_OPTS -cpu host,pmu=off"
  # TDX guest object with quote generation socket
  QEMU_OPTS="$QEMU_OPTS -object {\"qom-type\":\"tdx-guest\",\"id\":\"tdx0\",\"quote-generation-socket\":{\"type\":\"vsock\",\"cid\":\"2\",\"port\":\"4050\"}}"
  QEMU_OPTS="$QEMU_OPTS -machine q35,confidential-guest-support=tdx0,memory-backend=ram1,kernel-irqchip=split,hpet=off"
  # Use -bios for TDX boot
  QEMU_OPTS="$QEMU_OPTS -bios /usr/share/ovmf/OVMF.fd"
  # Disk boot (Ubuntu cloud image)
  QEMU_OPTS="$QEMU_OPTS -device virtio-net-pci,disable-legacy=on,iommu_platform=true,netdev=vmnic,romfile="
  QEMU_OPTS="$QEMU_OPTS -nodefaults"
  QEMU_OPTS="$QEMU_OPTS -nographic"
  QEMU_OPTS="$QEMU_OPTS -serial mon:stdio"
  QEMU_OPTS="$QEMU_OPTS -monitor pty"
else
  echo "Starting QEMU VM in regular mode (no CVM)..."
  QEMU_OPTS="$QEMU_OPTS -drive if=pflash,format=raw,unit=0,file=$OVMF_CODE,readonly=on"
  QEMU_OPTS="$QEMU_OPTS -drive if=pflash,format=raw,unit=1,file=$OVMF_VARS_COPY"
  QEMU_OPTS="$QEMU_OPTS -cpu host"
  QEMU_OPTS="$QEMU_OPTS -machine q35"
  QEMU_OPTS="$QEMU_OPTS -device virtio-net-pci,netdev=vmnic,romfile="
fi

# Execute QEMU (use eval to handle complex quoting)
echo "Full QEMU command:"
echo "$QEMU_CMD $QEMU_OPTS"
echo ""
$QEMU_CMD $QEMU_OPTS
