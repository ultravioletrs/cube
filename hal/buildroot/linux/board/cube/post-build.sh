#!/bin/sh
# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

set -u
set -e

# Add a console on tty1
if [ -e ${TARGET_DIR}/etc/inittab ]; then
    grep -qE '^tty1::' ${TARGET_DIR}/etc/inittab || \
	sed -i '/GENERIC_SERIAL/a\
tty1::respawn:/sbin/getty -L  tty1 0 vt100 # QEMU graphical window' ${TARGET_DIR}/etc/inittab
fi

# Configure DNS with direct nameservers (systemd-resolved has DNSSEC issues in VMs)
cat > ${TARGET_DIR}/etc/resolv.conf << 'EOF'
nameserver 8.8.8.8
nameserver 1.1.1.1
nameserver 8.8.4.4
nameserver 1.0.0.1
EOF

# Create the mount points for 9p shares
mkdir -p ${TARGET_DIR}/etc/cube/certs

# Ensure /etc/fstab exists
if [ ! -f "${TARGET_DIR}/etc/fstab" ]; then
    touch "${TARGET_DIR}/etc/fstab"
fi

# Add the 9p certificate mount entry to /etc/fstab
grep -q "certs_share /etc/cube/certs" ${TARGET_DIR}/etc/fstab || \
echo "certs_share /etc/cube/certs 9p trans=virtio,version=9p2000.L,cache=mmap 0 0" >> "${TARGET_DIR}/etc/fstab"
