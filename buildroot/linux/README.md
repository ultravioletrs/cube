# Hardware Abstraction Layer (HAL) for Confidential Computing

Cube HAL for Linux is framework for building custom in-enclave Linux distribution.

## Usage

HAL uses [Buildroot](https://buildroot.org/)'s [_External Tree_ mechanism](https://buildroot.org/downloads/manual/manual.html#outside-br-custom) for building custom distro:

```bash
git clone https://gitlab.com/buildroot.org/buildroot.git
cd buildroot
git checkout 2023.08
make BR2_EXTERNAL=../cube/buildroot/linux cube_defconfig
# Execute 'make menuconfig' only if you want to make additional configuration changes to Buildroot.
make menuconfig
make
```
