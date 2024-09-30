#!/bin/sh

# IFACES are all network interfaces excluding lo (LOOPBACK) and sit interfaces
IFACES=$(ip link show | grep -vE 'LOOPBACK|sit*' | awk -F': ' '{print $2}')

# This for loop brings up all network interfaces in IFACES and dhclient obtains an IP address for the every interface
for IFACE in $IFACES; do
    STATE=$(ip link show $IFACE | grep DOWN)
    if [ -n "$STATE" ]; then
        ip link set $IFACE up
    fi

    IP_ADDR=$(ip addr show $IFACE | grep 'inet ')
    if [ -z "$IP_ADDR" ]; then
        dhclient $IFACE
    fi
done

# Mount filesystem
mkdir -p /mnt/docker-volume
mount /dev/vda /mnt/docker-volume

# Change the docker.service file to allow the Docker to run in RAM
mkdir -p /etc/systemd/system/docker.service.d

# Create or overwrite the override.conf file with the new Environment variable
tee /etc/systemd/system/docker.service.d/override.conf > /dev/null <<EOF
[Service]
Environment=DOCKER_RAMDISK=true
EOF

systemctl daemon-reload

NUM_OF_PERMITED_IFACE=1

NUM_OF_IFACE=$(ip route | grep -Eo 'dev [a-z0-9]+' | awk '{ print $2 }' | sort | uniq | wc -l)

if [ $NUM_OF_IFACE -gt $NUM_OF_PERMITED_IFACE ]; then
    echo "More then one network interface in the VM"
    exit 1
fi

DEFAULT_IFACE=$(route | grep '^default' | grep -o '[^ ]*$')
