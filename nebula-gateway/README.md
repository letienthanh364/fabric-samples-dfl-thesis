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
STATE_PEER_ROUTES="state-1=peer0|peer1,state-2=peer1|peer2" \
AUTH_JWT_SECRET="replace-me" \
DOCKER_BUILDKIT=1 docker compose up --build
```

The compose file also honours variables defined in a `.env` file or your shell environment, so you can export them once instead of prefixing every command.

The compose file brings up the Fabric network, configures the channel (`nebulachannel` by default), deploys the `basic` chaincode, and finally starts the REST gateway on port `9000`. The gateway process now waits until the channel exists and peers have joined before binding the HTTP port, so you won’t accidentally hit it while the network is still bootstrapping.

Stop and clean up with:

```bash
docker compose down -v
```

## Gateway environment variables

| Variable | Default in compose | Purpose |
| --- | --- | --- |
| `FABRIC_CHANNEL` | `nebulachannel` | Fabric channel the gateway targets |
| `FABRIC_CHAINCODE` | `basic` | Chaincode name |
| `MSP_ID` | `Org1MSP` | MSP ID used by the gateway identity |
| `ORG_CRYPTO_PATH` | `/organizations/peerOrganizations/org1.nebula.com` | Base path for mounted MSP material |
| `ADMIN_IDENTITY` | `Admin@org1.nebula.com` | Gateway signing identity |
| `ORDERER_ENDPOINT` | `orderer.nebula.com:7050` | Orderer endpoint used for submits |
| `ORDERER_TLS_CA` | `/organizations/ordererOrganizations/.../tlsca.nebula.com-cert.pem` | Orderer TLS CA |
| `ORG_DOMAIN` | `org1.nebula.com` | Used to derive peer TLS paths |
| `PEER_ENDPOINTS` | `peer0=...7051,peer1=...8051,peer2=...9051` | Map of peer names to gRPC endpoints |
| `FABRIC_CFG_PATH` | `/etc/hyperledger/fabric` | Fabric config path for CLI |
| `STATE_PEER_ROUTES` | `state-1=peer0\|peer1,state-2=peer1\|peer2` | Route table mapping each state to ≥2 peers (round-robin selection) |
| `AUTH_JWT_SECRET` | `replace-me` (development only) | Shared HS256 secret used to validate JWTs |

Override any value by exporting it before `docker compose up` or adding it to `.env`.

## API Gateway

Base URL: `http://localhost:9000`

All Fabric-backed endpoints require a Bearer token and the gateway selects the peer internally using the `state` claim embedded in the JWT, so the caller never chooses the peer explicitly.

### Authentication & authorization

* Every request must include `Authorization: Bearer <JWT>`. The token must be signed with the shared `AUTH_JWT_SECRET` using HS256 and include `sub`, `state`, `role`, and `exp` claims (Unix seconds).
* Allowed roles: `trainer`, `aggregator`, `admin`. All roles can `GET` the genesis model endpoints while only `admin` is allowed to `POST`.
* The gateway consults `STATE_PEER_ROUTES` to decide which Fabric peer to hit for a given `state`, using round-robin across the configured list.

To mint a test token once you know `AUTH_JWT_SECRET`:

```bash
SECRET="replace-me" node -e '
const crypto=require("crypto");
const b=v=>Buffer.from(JSON.stringify(v)).toString("base64url");
const header={alg:"HS256",typ:"JWT"};
const payload={sub:"node-123",state:"state-1",role:"admin",exp:Math.floor(Date.now()/1000)+3600};
const unsigned=`${b(header)}.${b(payload)}`;
const sig=crypto.createHmac("sha256",process.env.SECRET).update(unsigned).digest("base64url");
console.log(`${unsigned}.${sig}`);
'
```

Use the output as the Bearer token in Postman/cURL.

### Health

```
GET /health
```

Response:

```json
{"status":"ok","defaultPeer":"peer0"}
```

### List assets

```
GET /assets
```

Returns the JSON array from the `GetAllAssets` chaincode function running on the selected peer.

### Create an asset

```
POST /assets
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
POST /job-contract/genesis-model-cid
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

`GET /job-contract/genesis-model-cid?jobId=credit-risk-v1` returns the metadata that was recorded for that job, including the timestamp of the last update:

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
POST /job-contract/genesis-model-hash
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

`GET /job-contract/genesis-model-hash?jobId=credit-risk-v1` returns the stored verification material:

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

For the full list of job contract endpoints (including the training configuration API) and their request/response schemas, see `api/internal/jobcontract/README.md`.

## Customisation

- Adjust MSP/crypto paths or Fabric defaults (such as `FABRIC_CHANNEL`) via environment variables in `docker-compose.yaml`. Whatever channel name you choose must match the one created by the bootstrap container.
- The `scripts/bootstrap.sh` flow can be tuned (channel name, chaincode label, etc.) through the `CHANNEL_NAME`, `CHAINCODE_*`, and related variables.
- Replace `chaincode/asset-transfer-basic` with your own implementation by updating the mounted path and chaincode metadata.

With this structure you can treat `docker compose up` as the single entry point for provisioning the Fabric network and the Nebula API gateway in one go.

## Redeploying code without rebuilding the whole stack

### Chaincode updates

Each time you change the Go chaincode you need to bump the chaincode version/sequence so Fabric will accept the new package. Rebuilding the API container alone is not enough; you must re-run the bootstrap script inside the CLI container so the peers install and approve the new definition. You can do this live without tearing down the network.

To check what is currently committed on the channel:

```bash
cd nebula-gateway
docker exec nebula-cli bash -c \
  'peer lifecycle chaincode querycommitted --channelID nebulachannel --name basic'
```

Fabric will print the `Version` label and `Sequence` number. When you redeploy, increment the sequence by one (and pick a matching version label so the package name stays consistent), then rerun the bootstrap script:

```bash
# example: move from v1.2/sequence 3 to v1.3/sequence 4
docker exec nebula-cli bash -c \
  'CHAINCODE_VERSION=1.3 CHAINCODE_SEQUENCE=4 /scripts/bootstrap.sh'
```

The `bootstrap.sh` script reuses the existing channel and orderer, repackages the code into `chaincode/<name>_<version>.tar.gz`, installs it on all peers, approves, and commits the definition with the supplied version/sequence. Update the numbers again for the next change—run this command every time you modify anything under `chaincode/asset-transfer-basic`, otherwise the network will keep running the older chaincode and the gateway will report “function not found” errors for new transactions.

### API gateway rebuilds

You do not need to stop the network to pick up HTTP API changes. Rebuild the gateway image and restart only that service:

```bash
docker compose build api-gateway
docker compose up -d api-gateway
```

Alternatively `docker compose restart api-gateway` is enough for changes that do not require rebuilding the binary.
