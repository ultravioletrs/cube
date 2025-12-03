---
description: Generate attestation policy for Cube AI CVMs
---

# Attestation Policy Generation for Cube AI CVMs

This guide explains how to generate attestation policies for Cube AI Confidential Virtual Machines (CVMs) after launching them. The attestation policy is generated **once** when the CVM is created and is used to verify the integrity and authenticity of the CVM during attestation.

## Prerequisites

- **Cocos CLI** installed and built (`cocos-cli`)
- Access to the CVM to retrieve attestation reports
- Appropriate permissions to access cloud provider attestation services (for GCP/Azure)

## Platform-Specific Instructions

### GCP (Google Cloud Platform) - SEV-SNP

GCP CVMs use AMD SEV-SNP with vTPM attestation.

#### Step 1: Obtain vTPM Attestation Report

SSH into your GCP CVM and retrieve the vTPM attestation report:

```bash
# Inside the CVM
sudo cat /sys/kernel/security/tpm0/binary_bios_measurements > attestation.bin
```

Copy the `attestation.bin` file to your local machine where Cocos CLI is installed.

#### Step 2: Generate Attestation Policy

Run the Cocos CLI command to generate the policy:

```bash
cocos-cli policy gcp attestation.bin <vcpu_count>
```

**Parameters:**
- `attestation.bin`: Path to the vTPM attestation report file
- `<vcpu_count>`: Number of vCPUs allocated to the CVM (e.g., 4, 8, 16)

**Optional flags:**
- `--json`: Use if the attestation report is in JSON format instead of binary

**Output:**
- `attestation_policy.json`: Generated attestation policy file

#### Step 3: Verify the Policy

The generated policy includes:
- Measurement values extracted from the attestation report
- Launch endorsement from Google's attestation service
- vCPU count configuration

---

### Azure - SEV-SNP

Azure CVMs use AMD SEV-SNP with Microsoft Azure Attestation (MAA).

#### Step 1: Obtain MAA Token

SSH into your Azure CVM and retrieve the MAA token:

```bash
# Inside the CVM
# Use Azure's attestation client to get the MAA token
# Save the token to a file
echo "<maa_token>" > maa_token.txt
```

Alternatively, use Azure CLI or SDK to retrieve the attestation token.

Copy the `maa_token.txt` file to your local machine.

#### Step 2: Generate Attestation Policy

Run the Cocos CLI command:

```bash
cocos-cli policy azure maa_token.txt <product_name>
```

**Parameters:**
- `maa_token.txt`: Path to the MAA token file
- `<product_name>`: AMD product name (e.g., `Milan`, `Genoa`)

**Optional flags:**
- `--policy <value>`: Guest policy value (default: 196639)

**Output:**
- `attestation_policy.json`: Generated attestation policy file

---

### Generic SEV-SNP (On-Premises or Other Clouds)

For generic SEV-SNP platforms not using GCP or Azure.

#### Step 1: Obtain Attestation Report

Retrieve the SEV-SNP attestation report from your CVM:

```bash
# Inside the CVM
# Method depends on your platform
# Example for direct SEV-SNP access:
sudo cat /dev/sev-guest > attestation_report.bin
```

#### Step 2: Generate Attestation Policy

Use the Rust-based script in Cocos:

```bash
cd /path/to/cocos-ai/scripts/attestation_policy/sev-snp
make

# Run the binary
cd target/release
./attestation_policy --policy 196608 --pcr ../../pcr_values.json
```

**Parameters:**
- `--policy`: 64-bit policy value (default: 196608)
- `--pcr`: Path to PCR values JSON file (optional)

**Output:**
- `attestation_policy.json`: Generated attestation policy file

---

### Intel TDX

For Intel TDX-based CVMs.

#### Step 1: Obtain TDX Attestation Report

Retrieve the TDX quote from your CVM:

```bash
# Inside the CVM
# Method depends on your platform
# Example:
sudo cat /dev/tdx-guest > tdx_quote.bin
```

#### Step 2: Generate Attestation Policy

Run the Cocos CLI command:

```bash
cocos-cli policy tdx tdx_quote.bin [flags]
```

**Optional flags:**
- `--qe_vendor_id <hex>`: Expected QE_VENDOR_ID (16 bytes hex)
- `--mr_seam <hex>`: Expected MR_SEAM measurement (48 bytes hex)
- `--td_attributes <hex>`: Expected TD_ATTRIBUTES (8 bytes hex)
- `--xfam <hex>`: Expected XFAM (8 bytes hex)
- `--mr_td <hex>`: Expected MR_TD measurement (48 bytes hex)
- `--rtmrs <hex,hex,hex,hex>`: Comma-separated RTMR values (4 values, 48 bytes each)
- `--minimum_tee_tcb_svn <hex>`: Minimum TEE_TCB_SVN (16 bytes hex)
- `--minimum_qe_svn <value>`: Minimum QE_SVN
- `--minimum_pce_svn <value>`: Minimum PCE_SVN
- `--get_collateral`: Download necessary collaterals for additional checks

**Output:**
- `attestation_policy.json`: Generated attestation policy file

---

## Additional Policy Operations

After generating the base attestation policy, you may need to perform additional operations:

### Add Measurement to Policy

Update the measurement field in an existing policy:

```bash
cocos-cli policy measurement <base64_measurement> attestation_policy.json
```

**Parameters:**
- `<base64_measurement>`: Base64-encoded measurement value (48 bytes)
- `attestation_policy.json`: Path to the policy file to update

### Add Host Data to Policy

Update the host data field in an existing policy:

```bash
cocos-cli policy hostdata <base64_hostdata> attestation_policy.json
```

**Parameters:**
- `<base64_hostdata>`: Base64-encoded host data value (32 bytes)
- `attestation_policy.json`: Path to the policy file to update

### Extend PCR with Computation Manifest

Extend PCR16 with computation manifest hashes:

```bash
cocos-cli policy extend attestation_policy.json manifest1.json [manifest2.json ...]
```

**Parameters:**
- `attestation_policy.json`: Path to the policy file to update
- `manifest1.json`, `manifest2.json`, ...: Paths to computation manifest files

This command:
1. Computes SHA-256 and SHA-384 hashes of each manifest
2. Extends PCR16 with the manifest hashes
3. Updates the policy file with new PCR values

---

## Using the Policy with Cube AI

Once you have generated the `attestation_policy.json` file:

1. **Store the policy securely**: This file contains the expected measurements and configuration for your CVM.

2. **Configure Cube AI**: Provide the policy file to Cube AI's attestation verification system.

3. **Verify attestation**: When clients connect to your Cube AI CVM, they will use this policy to verify that the CVM is running the expected code in a genuine TEE.
---

## Configuring Cube Proxy

To enable attestation verification in the Cube Proxy, you must configure it to use the generated `attestation_policy.json` file.

### Proxy Configuration for GCP and Azure

On platforms like GCP and Azure, the agent cannot fetch attestation reports if the proxy is already enforcing attestation (aTLS or mTLS) because the policy hasn't been generated yet. You must follow a two-step process:

#### Step 1: Initial Launch (No-TLS)

1.  Configure the proxy to **disable** Attested TLS. This allows the agent to start without verifying the CVM's identity, which is necessary to fetch the initial attestation materials.

    ```yaml
    services:
      cube-proxy:
        environment:
          - UV_CUBE_AGENT_ATTESTED_TLS=false
    ```

2.  Start the services.

#### Step 2: Fetch Attestation Materials

Retrieve the necessary attestation data from the running CVM:

*   **GCP**: Fetch the vTPM attestation report (`attestation.bin`).
*   **Azure**: Fetch the MAA token (`maa_token.txt`) and the attestation report.

#### Step 3: Generate Policy

Generate the `attestation_policy.json` using the `cocos-cli` as described in the platform-specific sections above.

#### Step 4: Enable Attested TLS

Once the policy is generated:

1.  **Mount the Policy**: Update your configuration to mount the `attestation_policy.json` into the proxy container.

    ```yaml
    services:
      cube-proxy:
        volumes:
          - ./attestation_policy.json:/etc/cube/proxy/attestation_policy.json:ro
    ```

2.  **Enable aTLS**: Update the environment variables to point to the policy and enable Attested TLS (or Mutual aTLS).

    ```yaml
    services:
      cube-proxy:
        environment:
          - UV_CUBE_AGENT_ATTESTATION_POLICY=/etc/cube/proxy/attestation_policy.json
          - UV_CUBE_AGENT_ATTESTED_TLS=true  # or 'mutual' for Mutual aTLS
    ```

3.  **Restart Proxy**: Restart the `cube-proxy` service to apply the changes. Now, all connections to the agent will be verified against your policy.

---

## Troubleshooting

### Policy Generation Fails

- **Check attestation report format**: Ensure the report file is in the correct format (binary or JSON)
- **Verify cloud provider access**: For GCP/Azure, ensure you have network access to attestation services
- **Check file permissions**: Ensure you have read access to attestation report files

### Invalid Measurements

- **Verify CVM configuration**: Ensure the CVM is launched with the expected firmware and configuration
- **Check for updates**: Firmware updates may change measurement values
- **Regenerate policy**: If the CVM configuration changes, regenerate the policy

### PCR Extension Issues

- **Manifest format**: Ensure manifest files are valid JSON
- **File access**: Verify all manifest files are readable
- **PCR values**: Check that initial PCR values in the policy are correct

## Additional Resources

- [Cocos AI Documentation](https://docs.cocos.ultraviolet.rs)
- [Cocos CLI Documentation](https://docs.cocos.ultraviolet.rs/cli)
- [AMD SEV-SNP Documentation](https://www.amd.com/en/developer/sev.html)
- [Intel TDX Documentation](https://www.intel.com/content/www/us/en/developer/tools/trust-domain-extensions/overview.html)
- [Google Confidential Computing](https://cloud.google.com/confidential-computing)
- [Azure Confidential Computing](https://azure.microsoft.com/en-us/solutions/confidential-compute/)
