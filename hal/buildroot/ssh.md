# SSH Configuration For Cube AI

## Overview

Cube AI uses a SSH to enable remote access to the VM. By default, the SSH server does not allow root login. You can either enable root login or create a new user with a specific password. This also allows you to copy files to the VM.

## Add new user

```bash
adduser --gecos "[your name]" --shell /bin/bash <username>
```

For example:

```bash
adduser --gecos "Rodney Osodo" --shell /bin/bash rodneyosodo
```

## Enable SSH root login

Edit the `/etc/ssh/sshd_config` file and add the following line:

```bash
PermitRootLogin yes
```

Save the file and restart the SSH server:

```bash
systemctl restart sshd
```

## To copy files to the VM

```bash
scp -P 6190 <local_file> <username>@<vm_ip>:<remote_file>
```

For example:

```bash
scp -P 6190 test.txt rodneyosodo@localhost:/home/rodneyosodo
```
