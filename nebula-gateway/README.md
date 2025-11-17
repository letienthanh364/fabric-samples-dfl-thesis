# Nebula Gateway Stack

This folder contains a self-contained Hyperledger Fabric deployment with three peer nodes, a single Raft orderer, channel artifacts, and an HTTP API gateway that proxies basic asset operations from NEBULA DFL nodes to the blockchain network. Everything is orchestrated with a single `docker compose` file so that the Fabric network and the gateway can be brought online (or torn down) together.

## Contents

- `organizations/`: cryptographic material generated with `cryptogen` for one orderer org and one peer org that hosts 3 peers.
- `system-genesis-block/` and `channel-artifacts/`: ready-to-use genesis block and channel transaction (channel name `nebulachannel`).
- `chaincode/asset-transfer-basic/`: Go chaincode (vendor directory included) that implements Fabric's asset-transfer sample and avoids fetching dependencies from the internet.
- `scripts/bootstrap.sh`: idempotent lifecycle script executed by the CLI container to create the channel, join all peers, package/install/approve/commit chaincode, and seed the ledger.
- `api/`: lightweight Go HTTP service that shells out to the Fabric `peer` CLI to submit/evaluate transactions on behalf of clients. The `peer` query parameter lets every request specify which peer should endorse the transaction.
- `docker-compose.yaml`: orchestrates 1 orderer, 3 peers, the bootstrap CLI, and the API gateway.

## Usage

From the repository root:

```bash
cd nebula-gateway
DOCKER_BUILDKIT=1 docker compose up --build
```

The compose file brings up the Fabric network, configures the channel (`nebulachannel` by default), deploys the `basic` chaincode, and finally starts the REST gateway on port `9000`. The gateway process now waits until the channel exists and peers have joined before binding the HTTP port, so you won’t accidentally hit it while the network is still bootstrapping.

Stop and clean up with:

```bash
docker compose down -v
```

## API Gateway

Base URL: `http://localhost:9000`

Every GET/POST request accepts a `peer` query parameter (`peer0`, `peer1`, `peer2`). If omitted or unknown, `peer0` is used. Each peer has its own TLS root cert mounted into the container so the gateway can target a specific NEBULA handler when committing data.

### Health

```
GET /health?peer=peer1
```

Response:

```json
{"status":"ok","peer":"peer1"}
```

### List assets

```
GET /assets?peer=peer2
```

Returns the JSON array from the `GetAllAssets` chaincode function running on the selected peer.

### Create an asset

```
POST /assets?peer=peer0
Content-Type: application/json

{
  "id": "asset-42",
  "color": "green",
  "size": 16,
  "owner": "nebula-client-a",
  "appraisedValue": 900
}
```

The gateway forwards the invocation to the requested peer and orderer using TLS and waits for the commit event before returning `201 Created`.

### Job contract – genesis model CID

```
POST /job-contract/genesis-model-cid?peer=peer0
Content-Type: application/json

{
  "jobId": "credit-risk-v1",
  "cid": "bafybeiemszxtnmn2a...",
  "purpose": "credit-risk-classifier",
  "modelFamily": "xgboost",
  "datasetSummary": "Normalized credit records (v3)",
  "notes": "Untrained weights shipped by provider ACME"
}
```

`GET /job-contract/genesis-model-cid?jobId=credit-risk-v1&peer=peer2` returns the metadata that was recorded for that job, including the timestamp of the last update:

```json
{
  "jobId": "credit-risk-v1",
  "cid": "bafybeiemszxtnmn2a...",
  "purpose": "credit-risk-classifier",
  "modelFamily": "xgboost",
  "datasetSummary": "Normalized credit records (v3)",
  "notes": "Untrained weights shipped by provider ACME",
  "updatedAt": "2024-04-05T11:03:10.317Z"
}
```

### Job contract – genesis model hash

```
POST /job-contract/genesis-model-hash?peer=peer1
Content-Type: application/json

{
  "jobId": "credit-risk-v1",
  "hash": "9b7f4dfa4f2a...",
  "hashAlgorithm": "sha256",
  "modelFormat": "onnx",
  "compression": "gzip",
  "notes": "Hash of bafybeiemszxtnmn2a..."
}
```

`GET /job-contract/genesis-model-hash?jobId=credit-risk-v1&peer=peer1` returns the stored verification material:

```json
{
  "jobId": "credit-risk-v1",
  "hash": "9b7f4dfa4f2a...",
  "hashAlgorithm": "sha256",
  "modelFormat": "onnx",
  "compression": "gzip",
  "notes": "Hash of bafybeiemszxtnmn2a...",
  "updatedAt": "2024-04-05T11:03:10.317Z"
}
```

## Customisation

- Adjust MSP/crypto paths or Fabric defaults (such as `FABRIC_CHANNEL`) via environment variables in `docker-compose.yaml`. Whatever channel name you choose must match the one created by the bootstrap container.
- The `scripts/bootstrap.sh` flow can be tuned (channel name, chaincode label, etc.) through the `CHANNEL_NAME`, `CHAINCODE_*`, and related variables.
- Replace `chaincode/asset-transfer-basic` with your own implementation by updating the mounted path and chaincode metadata.

With this structure you can treat `docker compose up` as the single entry point for provisioning the Fabric network and the Nebula API gateway in one go.

## Redeploying code without rebuilding the whole stack

### Chaincode updates

Each time you change the Go chaincode you need to bump the chaincode version/sequence so Fabric will accept the new package. You can do this live without tearing down the network:

```bash
# example: move from v1.0/sequence 1 to v1.1/sequence 2
docker exec nebula-cli bash -c \
  'CHAINCODE_VERSION=1.1 CHAINCODE_SEQUENCE=2 /scripts/bootstrap.sh'
```

The `bootstrap.sh` script reuses the existing channel and orderer, repackages the code into `chaincode/<name>_<version>.tar.gz`, installs it on all peers, approves, and commits the definition with the supplied version/sequence. Update the numbers again for the next change.

### API gateway rebuilds

You do not need to stop the network to pick up HTTP API changes. Rebuild the gateway image and restart only that service:

```bash
docker compose build api-gateway
docker compose up -d api-gateway
```

Alternatively `docker compose restart api-gateway` is enough for changes that do not require rebuilding the binary.
