# Hardware Abstraction Layer (HAL) for Confidential Computing

Cube HAL for Linux is framework for building custom in-enclave Linux distribution.

## Usage

HAL uses [Buildroot](https://buildroot.org/)'s [_External Tree_ mechanism](https://buildroot.org/downloads/manual/manual.html#outside-br-custom) for building custom distro:

### Step 1: Clone the Buildroot Repository

First, download the Buildroot source code from its repository.

```bash
git clone https://gitlab.com/buildroot.org/buildroot.git
```

### Step 2: Clone the Cube Project Repository

Next, download the Cube project.

```bash
git clone https://github.com/ultravioletrs/cube.git
```

### Step 3: Go to the Buildroot Directory

```bash
cd buildroot
```

### Step 4: Set Up the Build Configuration

Run the following command to configure Buildroot with settings from the Cube project.

```bash
make BR2_EXTERNAL=../cube/hal/buildroot/linux cube_defconfig
```

- `BR2_EXTERNAL=../cube/hal/buildroot/linux`: Tells Buildroot to use the external configuration files from the Cube project located at `../cube/hal/buildroot/linux`.
- `cube_defconfig`: Loads the default configuration for Cube. This sets up Buildroot with settings that are specific to the Cube project.

### Step 5 (Optional): Make Additional Configuration Changes

If you want to adjust any settings manually, you can run the following command to customize the Buildroot configuration:

```bash
make menuconfig
```

This step is optional. If you don’t need to make any changes, you can skip it.

### Step 6: Build the Project

Finally, build the project by running:

```bash
make
```
