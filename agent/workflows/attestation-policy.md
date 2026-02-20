---
description: Generate attestation policy for Cube AI CVMs
---

# Attestation Policy Generation for Cube AI CVMs

This guide explains how to generate attestation policies for Cube AI Confidential Virtual Machines (CVMs) after launching them. The attestation policy is generated **once** when the CVM is created and is used to verify the integrity and authenticity of the CVM during attestation.

## Prerequisites

- **Cocos CLI** installed and built (`cocos-cli`)
- A running Cube deployment accessible through the **Traefik** reverse proxy (`https://localhost/proxy` for local `make up`)
- A valid SuperMQ access token and domain ID for authenticated Cube Proxy API calls
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

For this repository's local Docker flow:
- `make up` already runs `make disable-atls`, so initial attestation-policy bootstrapping works out of the box.
- If services are already running with aTLS enabled, run `make disable-atls && docker compose -f docker/compose.yaml up -d`.

## Retrieving Attestation Reports via the Cube Proxy API

All platforms use the same Cube Proxy API endpoint to retrieve attestation reports. The agent automatically detects the TEE platform (TDX, SEV-SNP, SNPvTPM, Azure) at startup - there is no `attestation_type` field in the request.

Use the Traefik URL, not the internal container port:
- Local: `https://localhost/proxy`
- Cloud: `https://<your-domain>/proxy`

**Endpoint:** `POST <traefik_base_url>/<domain_id>/attestation`

**Request body:**

| Field | Type | Description |
|-------|------|-------------|
| `report_data` | string | Base64-encoded data to embed in the report (max 64 bytes). Use `""` for empty. |
| `nonce` | string | Base64-encoded nonce (max 32 bytes). Use `""` for empty. |
| `to_json` | boolean | Keep `false` for policy generation inputs (`attestation.bin`). Use `true` only when you need a human-readable report for inspection. |

**Example — JSON response:**

```bash
curl -ksSf -X POST https://<traefik-host>/proxy/<domain_id>/attestation \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"report_data": "", "nonce": "", "to_json": true}' \
  -o attestation.json
```

**Example — Binary response:**

```bash
curl -ksSf -X POST https://<traefik-host>/proxy/<domain_id>/attestation \
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

For detailed `cocos-cli` usage, refer to the [Cocos CLI documentation](https://docs.cocos.ultraviolet.rs/cli). The sections below cover the Cube-specific retrieval steps and the corresponding `cocos-cli policy` commands.

### GCP (Google Cloud Platform) - SEV-SNP

GCP CVMs use AMD SEV-SNP with vTPM attestation.

#### Step 1: Obtain vTPM Attestation Report

Retrieve the vTPM attestation report from the GCP CVM via the Cube Proxy API:

```bash
curl -ksSf -X POST https://<traefik-host>/proxy/<domain_id>/attestation \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"report_data": "", "nonce": "", "to_json": false}' \
  -o attestation.bin
```

`cocos-cli policy gcp` expects binary protobuf by default (`attestation.bin`).
If you retrieved JSON (`"to_json": true`), pass `--json`:

```bash
cocos-cli policy gcp --json attestation.json <vcpu_count>
```

#### Step 2: Generate Attestation Policy

Run the Cocos CLI command to generate the policy:

```bash
cocos-cli policy gcp attestation.bin <vcpu_count>
```

**Parameters:**
- `attestation.bin`: Path to the vTPM attestation report file (binary protobuf)
- `<vcpu_count>`: Number of vCPUs allocated to the CVM (e.g., 4, 8, 16)

**Optional flags:**
- `--json`: Parse input as JSON attestation instead of binary protobuf

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

The `cocos-cli policy azure` command requires a Microsoft Azure Attestation (MAA) JWT token. This token is not available through the Cube Proxy REST API (which returns a vTPM + SNP protobuf).

Generate the MAA JWT **on the Azure CVM itself** using Microsoft's [CVM guest attestation sample app](https://github.com/Azure/confidential-computing-cvm-guest-attestation/tree/main/cvm-attestation-sample-app). Follow the build and usage instructions in that repository's README, then run:

```bash
# On the Azure CVM (after building the sample app)
sudo ./AttestationClient -o token > azure_attest_token.jwt
```

Copy `azure_attest_token.jwt` to the machine where you will run `cocos-cli`.

#### Step 2: Generate Attestation Policy

`cocos-cli policy azure` parses the JWT offline and extracts SEV-SNP policy fields from the `x-ms-isolation-tee` claims:

```bash
cocos-cli policy azure azure_attest_token.jwt <product_name>
```

`cocos-cli` validates the JWT signature by fetching the MAA JSON Web Key Set (JWKS) over HTTPS, so the machine running this command needs outbound HTTPS access to the MAA endpoint (defaults to `https://sharedeus2.eus2.attest.azure.net`).

**Parameters:**
- `azure_attest_token.jwt`: Path to the MAA token file (JWT string)
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
curl -ksSf -X POST https://<traefik-host>/proxy/<domain_id>/attestation \
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
curl -ksSf -X POST https://<traefik-host>/proxy/<domain_id>/attestation \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"report_data": "", "nonce": "", "to_json": false}' \
  -o attestation.bin
```

Prefer keeping `"to_json": false` for policy generation inputs.

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
    curl -ksSf -X POST https://<traefik-host>/proxy/attestation/policy \
      -H "Authorization: Bearer <access_token>" \
      -H "Content-Type: application/json" \
      -d @attestation_policy.json
    ```

    A `201 Created` response confirms the policy was stored successfully. Each upload creates a new version; the proxy always serves the latest.

3. **Retrieve the policy**: The current attestation policy can be fetched via the Cube Proxy API:

    ```bash
    curl -ksSf -X GET https://<traefik-host>/proxy/<domain_id>/attestation/policy \
      -H "Authorization: Bearer <access_token>"
    ```

    This returns the raw attestation policy JSON. The UI also uses this endpoint to display the policy.

4. **Verify attestation**: When clients connect to your Cube AI CVM, the proxy uses this policy to verify that the CVM is running the expected code in a genuine TEE.

---

## Configuring Cube Proxy

To enable attestation verification in the Cube Proxy, you must configure it to use the generated attestation policy.

#### Step 1: Mount the Policy

Mount `attestation_policy.json` into the proxy container.
For this repository's local Compose files, the default mount target is `/etc/cube/attestation-policy.json`:

```yaml
services:
  cube-proxy:
    volumes:
      - ./attestation_policy.json:/etc/cube/attestation-policy.json:ro
```

#### Step 2: Enable Attested TLS

Update the environment variables to point to the mounted policy path and enable Attested TLS (or Mutual aTLS):

```yaml
services:
  cube-proxy:
    environment:
      - UV_CUBE_AGENT_ATTESTATION_POLICY=/etc/cube/attestation-policy.json
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
