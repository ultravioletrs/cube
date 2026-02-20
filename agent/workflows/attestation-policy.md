---
description: Generate attestation policy for Cube AI CVMs
---

# Attestation Policy Generation for Cube AI CVMs

This guide explains how to generate attestation policies for Cube AI Confidential Virtual Machines (CVMs) after launching them. The attestation policy is generated **once** when the CVM is created and is used to verify the integrity and authenticity of the CVM during attestation.

## Prerequisites

- **Cocos CLI** installed and built (`cocos-cli`)
- A running Cube deployment with the **Cube Proxy** accessible
- Valid access token and domain ID for the Cube Proxy API
- Appropriate permissions to access cloud provider attestation services (for GCP/Azure)

## Important: Initial Setup Without Attested TLS

When generating an attestation policy for the first time, the Cube Proxy **must not** enforce Attested TLS (aTLS). This is because aTLS requires a policy to verify the agent, but you need to connect to the agent first to obtain the attestation report used to generate that policy.

Before retrieving attestation reports, ensure the proxy is configured with aTLS disabled:

```yaml
services:
  cube-proxy:
    environment:
      - UV_CUBE_AGENT_ATTESTED_TLS=false
```

Start the services with this configuration. After generating and uploading the policy, you can enable aTLS as described in [Configuring Cube Proxy](#configuring-cube-proxy).

## Retrieving Attestation Reports via the Cube Proxy API

All platforms use the same Cube Proxy API endpoint to retrieve attestation reports. The agent automatically detects the TEE platform (TDX, SEV-SNP, SNPvTPM, Azure) at startup — there is no `attestation_type` field in the request.

**Endpoint:** `POST /<domain_id>/attestation`

**Request body:**

| Field | Type | Description |
|-------|------|-------------|
| `report_data` | string | Base64-encoded data to embed in the report (max 64 bytes). Use `""` for empty. |
| `nonce` | string | Base64-encoded nonce (max 32 bytes). Use `""` for empty. |
| `to_json` | boolean | `true` for JSON response, `false` for binary protobuf. |

**Example — JSON response:**

```bash
curl -sSf -X POST http://<proxy-host>:<proxy-port>/<domain_id>/attestation \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"report_data": "", "nonce": "", "to_json": true}' \
  -o attestation.json
```

**Example — Binary response:**

```bash
curl -sSf -X POST http://<proxy-host>:<proxy-port>/<domain_id>/attestation \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"report_data": "", "nonce": "", "to_json": false}' \
  -o attestation.bin
```

The response format depends on the platform the agent is running on:

| Platform | Response Content | Format |
|----------|-----------------|--------|
| **GCP (SNPvTPM)** | vTPM attestation with embedded SEV-SNP report | Protobuf (`attest.Attestation`) |
| **Azure** | vTPM attestation with embedded SEV-SNP report | Protobuf (`attest.Attestation`) |
| **Generic SEV-SNP** | Raw SEV-SNP attestation report | Binary SEV-SNP report |
| **Intel TDX** | Raw TDX quote | Binary TDX quote |

---

## Platform-Specific Instructions

### GCP (Google Cloud Platform) - SEV-SNP

GCP CVMs use AMD SEV-SNP with vTPM attestation.

#### Step 1: Obtain vTPM Attestation Report

Retrieve the vTPM attestation report from the GCP CVM via the Cube Proxy API:

```bash
curl -sSf -X POST http://<proxy-host>:<proxy-port>/<domain_id>/attestation \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"report_data": "", "nonce": "", "to_json": false}' \
  -o attestation.bin
```

To get the report in JSON format instead, set `"to_json": true` and save to `attestation.json`.

#### Step 2: Generate Attestation Policy

Run the Cocos CLI command to generate the policy:

```bash
cocos-cli policy gcp attestation.bin <vcpu_count>
```

**Parameters:**
- `attestation.bin`: Path to the vTPM attestation report file (binary protobuf)
- `<vcpu_count>`: Number of vCPUs allocated to the CVM (e.g., 4, 8, 16)

**Optional flags:**
- `--json`: Use if the attestation report is in JSON format (i.e., retrieved with `"to_json": true`)

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

#### Step 1: Obtain the MAA Token

The `cocos-cli policy azure` command requires a Microsoft Azure Attestation (MAA) JWT token. This token is not available through the Cube Proxy REST API (which returns a vTPM + SNP protobuf). Instead, use `cocos-cli` to retrieve it directly from the agent:

```bash
# Generate a 32-byte hex nonce (or use all zeros)
TOKEN_NONCE=$(openssl rand -hex 32)

cocos-cli attestation get azure-token \
  --token "$TOKEN_NONCE" \
  --azurejwt
```

This saves the MAA JWT to `azure_attest_token.jwt`. The `cocos-cli` must be configured to connect to the agent (set `AGENT_GRPC_URL` or equivalent to the agent's address).

#### Step 2: Generate Attestation Policy

Run the Cocos CLI command:

```bash
cocos-cli policy azure azure_attest_token.jwt <product_name>
```

**Parameters:**
- `maa_token.txt`: Path to the MAA token file (JWT string)
- `<product_name>`: AMD product name (e.g., `Milan`, `Genoa`)

**Optional flags:**
- `--policy <value>`: Guest policy value (default: 196639)

**Output:**
- `attestation_policy.json`: Generated attestation policy file

---

### Generic SEV-SNP (On-Premises or Other Clouds)

For generic SEV-SNP platforms not using GCP or Azure.

#### Step 1: Obtain Attestation Report

Retrieve the SEV-SNP attestation report from the CVM via the Cube Proxy API:

```bash
curl -sSf -X POST http://<proxy-host>:<proxy-port>/<domain_id>/attestation \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"report_data": "", "nonce": "", "to_json": false}' \
  -o attestation.bin
```

#### Step 2: Generate Attestation Policy

Use the Rust-based attestation policy tool in the Cocos repository:

```bash
cd cocos/scripts/attestation_policy/sev-snp
make

cd target/release
./attestation_policy --policy 196608 --pcr ../../pcr_values.json
```

**Parameters:**
- `--policy`: 64-bit guest policy value (default: 196608)
- `--pcr`: Path to PCR values JSON file (optional)

**Output:**
- `attestation_policy.json`: Generated attestation policy file

---

### Intel TDX

For Intel TDX-based CVMs.

#### Step 1: Obtain TDX Attestation Report

Retrieve the TDX quote from the CVM via the Cube Proxy API:

```bash
curl -sSf -X POST http://<proxy-host>:<proxy-port>/<domain_id>/attestation \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"report_data": "", "nonce": "", "to_json": false}' \
  -o attestation.bin
```

To get the report in JSON format instead, set `"to_json": true` and save to `attestation.json`.

#### Step 2: Generate Attestation Policy

Run the Cocos CLI command:

```bash
cocos-cli policy tdx attestation.bin [flags]
```

**Optional flags** (from `cocos-cli policy tdx --help`):
- `--qe_vendor_id <hex>`: Expected QE_VENDOR_ID field (16 bytes hex, unchecked if unset)
- `--mr_seam <hex>`: Expected MR_SEAM field (48 bytes hex, unchecked if unset)
- `--td_attributes <hex>`: Expected TD_ATTRIBUTES field (8 bytes hex, unchecked if unset)
- `--xfam <hex>`: Expected XFAM field (8 bytes hex, unchecked if unset)
- `--mr_td <hex>`: Expected MR_TD field (48 bytes hex, unchecked if unset)
- `--mr_config_id <hex>`: Expected MR_CONFIG_ID field (48 bytes hex, unchecked if unset)
- `--mr_owner <hex>`: Expected MR_OWNER field (48 bytes hex, unchecked if unset)
- `--mr_config_owner <hex>`: Expected MR_OWNER_CONFIG field (48 bytes hex, unchecked if unset)
- `--rtmrs <hex,hex,hex,hex>`: Comma-separated expected RTMR values (4 strings, each 48 bytes hex, unchecked if unset)
- `--minimum_tee_tcb_svn <hex>`: Minimum TEE_TCB_SVN field (16 bytes hex, unchecked if unset)
- `--minimum_qe_svn <value>`: Minimum QE_SVN field (uint32)
- `--minimum_pce_svn <value>`: Minimum PCE_SVN field (uint32)
- `--trusted_root <paths>`: Comma-separated paths to PEM CA bundles for Intel TDX root certificates (uses embedded root certificate if unset)
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

2. **Upload to Cube Proxy**: Seed the policy into the Cube Proxy database using the API (requires super admin privileges):

    ```bash
    curl -sSf -X POST http://<proxy-host>:<proxy-port>/attestation/policy \
      -H "Authorization: Bearer <access_token>" \
      -H "Content-Type: application/json" \
      -d @attestation_policy.json
    ```

    A `201 Created` response confirms the policy was stored successfully. Each upload creates a new version; the proxy always serves the latest.

3. **Retrieve the policy**: The current attestation policy can be fetched via the Cube Proxy API:

    ```bash
    curl -sSf -X GET http://<proxy-host>:<proxy-port>/<domain_id>/attestation/policy \
      -H "Authorization: Bearer <access_token>"
    ```

    This returns the raw attestation policy JSON. The UI also uses this endpoint to display the policy.

4. **Verify attestation**: When clients connect to your Cube AI CVM, the proxy uses this policy to verify that the CVM is running the expected code in a genuine TEE.

---

## Configuring Cube Proxy

To enable attestation verification in the Cube Proxy, you must configure it to use the generated attestation policy.

#### Step 1: Mount the Policy

Update your configuration to mount the `attestation_policy.json` into the proxy container:

```yaml
services:
  cube-proxy:
    volumes:
      - ./attestation_policy.json:/etc/cube/proxy/attestation_policy.json:ro
```

#### Step 2: Enable Attested TLS

Update the environment variables to point to the policy and enable Attested TLS (or Mutual aTLS):

```yaml
services:
  cube-proxy:
    environment:
      - UV_CUBE_AGENT_ATTESTATION_POLICY=/etc/cube/proxy/attestation_policy.json
      - UV_CUBE_AGENT_ATTESTED_TLS=true  # or 'mutual' for Mutual aTLS
```

#### Step 3: Restart Proxy

Restart the `cube-proxy` service to apply the changes. Now, all connections to the agent will be verified against your policy.

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
