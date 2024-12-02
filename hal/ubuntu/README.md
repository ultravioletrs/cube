# Ubuntu

This directory contains the cloud-init configuration files for Cube AI.

## After the first boot

For local development, replace the following IP address entries in `docker/.env` with the IP address of the qemu virtual machine as follows:

```bash
 UV_CUBE_NEXTAUTH_URL=http://<ip-address>:${UI_PORT}
```
