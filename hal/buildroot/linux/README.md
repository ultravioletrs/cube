# Hardware Abstraction Layer (HAL) for Confidential Computing

Cube HAL for Linux is a framework for building custom in-enclave Linux distributions with integrated AI workload support.

## Overview

The Cube HAL provides a complete embedded Linux environment with:
- **Cube Agent**: Ultraviolet Cube Agent for managing AI workloads
- **LLM Backends**: Choice of Ollama or vLLM for local language model inference
- **Buildroot Integration**: Streamlined build process with external tree mechanism

## Quick Start

### Prerequisites

- Git
- Build essentials (gcc, make, etc.)
- At least 8GB RAM
- 20GB free disk space

### Step 1: Clone the Buildroot Repository

Download the Buildroot source code:

```bash
git clone https://gitlab.com/buildroot.org/buildroot.git
```

### Step 2: Clone the Cube Project Repository

Download the Cube project:

```bash
git clone https://github.com/ultravioletrs/cube.git
```

### Step 3: Navigate to Buildroot Directory

```bash
cd buildroot
```

### Step 4: Configure the Build

Load the default Cube configuration:

```bash
make BR2_EXTERNAL=../cube/hal/buildroot/linux cube_defconfig
```

This command:
- Sets `BR2_EXTERNAL` to point to Cube's external configuration at `../cube/hal/buildroot/linux`
- Loads `cube_defconfig` with preconfigured settings for Cube Agent and LLM backends

### Step 5: Customize Configuration (Optional)

To modify the default configuration, use menuconfig:

```bash
make menuconfig
```

#### Key Configuration Options

Navigate to **Target packages → Cube AI Services**:

##### Cube Agent Configuration
- **Instance ID**: Unique identifier (default: `cube-agent-01`)
- **Host**: Bind address (default: `0.0.0.0`)
- **Port**: Service port (default: `7001`)
- **Log Level**: debug, info, warn, or error
- **LLM Backend Selection**:
  - Ollama (default)
  - vLLM
  - Custom URL
- **Agent Environment**:
  - **OS Build**: (default `UVC`)
  - **OS Distro**: (default `UVC`)
  - **OS Type**: (default `UVC`)
  - **VMPL**: VM Privilege Level (default `2`)
- **Security & TLS**:
  - **CA URL**: URL of the Certificate Authority
  - **Attested TLS**: Enable/Disable (default `Enabled`)
  - **Server CA Certificates**: Path to file
  - **Server Certificate**: Path to file
  - **Server Key**: Path to file
  - **Client CA Certificates**: Path to file

##### Ollama Backend (if selected)
- **Install default models**: Automatically pulls llama3.2:3b, starcoder2:3b, nomic-embed-text:v1.5
- **Custom models**: Space-separated list (e.g., `llama2:7b mistral:7b codellama:13b`)
- **GPU Support**: Enable NVIDIA or AMD GPU acceleration

##### vLLM Backend (if selected)
- **Model**: HuggingFace model identifier (default: `microsoft/DialoGPT-medium`)
- **GPU Memory Utilization**: Fraction of GPU memory (0.0-1.0, default: 0.85)
- **Max Model Length**: Maximum sequence length (default: 1024)
- **Custom Model Path**: Optional local model directory

### Step 6: Build the Project

Start the build process:

```bash
make
```

Build time varies (typically 30-120 minutes depending on configuration and hardware).

## Output

After successful build, you'll find:
- **Kernel**: `output/images/bzImage`
- **Root filesystem**: `output/images/rootfs.ext4`
- **Full image**: `output/images/sdcard.img` (if applicable)

## Package Structure

The Cube HAL includes three main packages:

```
package/
├── cube-agent/          # Cube Agent service
│   ├── Config.in        # Configuration options
│   ├── cube-agent.mk    # Build instructions
│   ├── S95cube-agent    # SysV init script
│   └── cube-agent.service  # systemd unit
├── ollama/              # Ollama LLM backend
│   ├── Config.in
│   ├── ollama.mk
│   ├── S96ollama
│   └── ollama.service
└── vllm/                # vLLM LLM backend
    ├── Config.in
    ├── vllm.mk
    ├── S96vllm
    └── vllm.service
```

## Runtime Service Management

### SysV Init Systems

Services start automatically via init scripts:

```bash
# Check service status
/etc/init.d/S96ollama status
/etc/init.d/S95cube-agent status

# Manual control
/etc/init.d/S96ollama start|stop|restart
/etc/init.d/S95cube-agent start|stop|restart
```

### Systemd Systems

Manage services with systemctl:

```bash
# Check status
systemctl status ollama
systemctl status cube-agent

# Control services
systemctl start|stop|restart ollama
systemctl enable|disable cube-agent
```

## Configuration Files

Runtime configurations are stored in:
- **Cube Agent**: `/etc/cube/agent.env`
- **Ollama**: Environment in init scripts/systemd units
- **vLLM**: `/etc/vllm/vllm.env`

### Example: Cube Agent Environment

```bash
UV_CUBE_AGENT_LOG_LEVEL=info
UV_CUBE_AGENT_HOST=0.0.0.0
UV_CUBE_AGENT_PORT=7001
UV_CUBE_AGENT_INSTANCE_ID=cube-agent-01
UV_CUBE_AGENT_TARGET_URL=http://localhost:11434
```

## Model Management

### Ollama Models

Models are stored in `/var/lib/ollama/models/`:

```bash
# List available models
ollama list

# Pull additional models
ollama pull llama2:7b

# Remove models
ollama rm llama3.2:3b
```

### vLLM Models

Models are cached in `/var/lib/vllm/` and `/var/cache/vllm/`:
- Downloaded from HuggingFace on first run (if not using custom path)
- Custom models: Place in path specified during configuration

## Network Endpoints

Default service endpoints:
- **Cube Agent**: `http://localhost:7001`
- **Ollama**: `http://localhost:11434`
- **vLLM**: `http://localhost:8000`

## Advanced Configuration

### GPU Acceleration

For GPU support:
1. Enable in menuconfig under Ollama or vLLM settings
2. Ensure appropriate drivers are available (NVIDIA CUDA or AMD ROCm)
3. Verify GPU detection: `nvidia-smi` or `rocm-smi`

### Custom LLM Backend

To use a custom backend URL:
1. Select "Custom URL" in Cube Agent backend choice
2. Set target URL (e.g., `http://192.168.1.100:8000`)
3. Ensure the endpoint implements OpenAI-compatible API

### Adding Custom Models at Build Time

#### Ollama
Set in menuconfig:
```
BR2_PACKAGE_OLLAMA_CUSTOM_MODELS="llama2:7b mistral:7b"
```

#### vLLM
Set custom model path:
```
BR2_PACKAGE_VLLM_CUSTOM_MODEL_PATH=/path/to/local/model
```

## Troubleshooting

### Service Fails to Start

Check logs:
```bash
# SysV init
tail -f /var/log/messages

# systemd
journalctl -u cube-agent -f
journalctl -u ollama -f
```

### Model Download Issues

For Ollama:
```bash
# Manually pull model
ollama pull <model-name>

# Check available space
df -h /var/lib/ollama
```

For vLLM:
```bash
# Check HuggingFace cache
ls -lh /var/cache/vllm

# Verify network connectivity
curl https://huggingface.co
```

### Backend Connection Errors

Verify backend is running and accessible:
```bash
# Test Ollama
curl http://localhost:11434/api/tags

# Test vLLM
curl http://localhost:8000/v1/models
```

## Security Considerations

- Services run as dedicated users (`ollama`, `vllm`) with restricted permissions
- Systemd units include security hardening (NoNewPrivileges, PrivateTmp, ProtectSystem)
