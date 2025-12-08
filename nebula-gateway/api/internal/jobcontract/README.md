# Job Contract API

This module exposes HTTP endpoints that let DFL nodes read/write job metadata on the Fabric ledger via the Nebula gateway. All routes require a valid JWT (`Authorization: Bearer …`) and the gateway resolves the caller’s `state` claim to the peers it is allowed to use.

Base path: `/job-contract`

## Common Fields

All endpoints operate on a `jobId` that uniquely identifies the distributed training job. The Fabric chaincode stores each job’s metadata under that key, so repeated submissions with the same `jobId` overwrite the previous values.

| Field | Type | Description |
| --- | --- | --- |
| `jobId` | string | Unique identifier for the job. Required in all requests. |

## Authentication & Roles

| Endpoint | Method | Roles allowed |
| --- | --- | --- |
| `/genesis-model-cid` | `GET` | `trainer`, `aggregator`, `admin` |
| `/genesis-model-cid` | `POST` | `admin` |
| `/genesis-model-hash` | `GET` | `trainer`, `aggregator`, `admin` |
| `/genesis-model-hash` | `POST` | `admin` |
| `/training-config` | `GET` | `trainer`, `aggregator`, `admin` |
| `/training-config` | `POST` | `admin` |

## Endpoints

### `GET /job-contract/genesis-model-cid`

Returns the current IPFS/IPLD content identifier that points to the genesis model artifact for the specified job.

Query parameters:

| Name | Required | Description |
| --- | --- | --- |
| `jobId` | ✔ | Job identifier. |

Response:

```json
{
  "jobId": "credit-risk-v1",
  "cid": "bafybeiemszxt...",
  "purpose": "fraud-detection",
  "modelFamily": "xgboost",
  "datasetSummary": "Normalized credit records",
  "notes": "Untrained weights",
  "updatedAt": "2024-06-10T08:31:42.184Z"
}
```

### `POST /job-contract/genesis-model-cid` (admin only)

Registers or updates the genesis model CID for a job.

Body fields:

| Field | Required | Description |
| --- | --- | --- |
| `jobId` | ✔ | Job identifier. |
| `cid` | ✔ | Content identifier for the model artifact. |
| `purpose` | ✔ | High-level description of the model’s intended use. |
| `modelFamily` | ✔ | Family/architecture (e.g., `cnn`, `xgboost`). |
| `datasetSummary` | ✖ | Free-form notes about the dataset (stored verbatim). |
| `notes` | ✖ | Additional remarks or provenance. |

Returns `201 Created` with `{"jobId": "<id>"}` when the ledger update succeeds.

### `GET /job-contract/genesis-model-hash`

Fetches the canonical hash metadata used to validate the genesis model file.

Query parameters: `jobId` (required).

Response:

```json
{
  "jobId": "credit-risk-v1",
  "hash": "9b7f4dfa...",
  "hashAlgorithm": "sha256",
  "modelFormat": "onnx",
  "compression": "gzip",
  "notes": "Hash of bafybeiemszxt...",
  "updatedAt": "2024-06-10T08:32:02.582Z"
}
```

### `POST /job-contract/genesis-model-hash` (admin only)

Body fields:

| Field | Required | Description |
| --- | --- | --- |
| `jobId` | ✔ | Job identifier. |
| `hash` | ✔ | Hex/base64 digest string. |
| `hashAlgorithm` | ✔ | Hash function used (e.g., `sha256`). |
| `modelFormat` | ✔ | Serialized model format (e.g., `onnx`). |
| `compression` | ✖ | Compression applied to the model artifact. |
| `notes` | ✖ | Free-form description. |

### `GET /job-contract/training-config`

Returns the distributed training parameters for a job.

Query parameters: `jobId` (required).

Response example:

```json
{
  "jobId": "credit-risk-v1",
  "modelName": "credit-risk",
  "modelVersion": "1.0.0",
  "datasetUri": "s3://datasets/credit-risk-v3",
  "objective": "Binary classification of good vs bad",
  "description": "Heterogeneous tabular DFL job",
  "roundDurationSec": 300,
  "batchSize": 64,
  "learningRate": 0.05,
  "maxClusterRounds": 10,
  "maxStateRounds": 5,
  "alpha": 0.8,
  "updatedAt": "2024-06-10T08:33:10.125Z"
}
```

### `POST /job-contract/training-config` (admin only)

Publishes the training configuration that DFL participants should follow.

Body fields:

| Field | Required | Description |
| --- | --- | --- |
| `jobId` | ✔ | Job identifier. |
| `modelName` | ✔ | Human-readable model name. |
| `modelVersion` | ✖ | Semantic version or git tag for the architecture. |
| `datasetUri` | ✔ | URI/locator of the dataset specification. |
| `objective` | ✔ | What the model is optimizing for (text). |
| `description` | ✖ | Additional context for the run. |
| `roundDurationSec` | ✔ | Target length of a training round in seconds (must be >0). |
| `batchSize` | ✔ | Mini-batch size per client (must be >0). |
| `learningRate` | ✔ | Base learning rate used in each round (must be >0). |
| `maxClusterRounds` | ✔ | Maximum rounds inside a cluster before aggregation (must be >0). |
| `maxStateRounds` | ✔ | Maximum rounds across state-level aggregations (must be >0). |
| `alpha` | ✔ | Alpha coefficient for dynamic step-size control (must be >0). |

A successful upsert returns `201 Created` and echoes the `jobId`.

## Error Handling

All validation errors return `400 Bad Request` with a JSON body such as:

```json
{
  "error": "roundDurationSec must be greater than zero"
}
```

Missing/invalid JWTs yield `401 Unauthorized`; callers with insufficient roles receive `403 Forbidden`.

## Testing the Endpoints

Use `curl` with a valid token:

```bash
TOKEN="..." # Bearer token signed with AUTH_JWT_SECRET

curl -H "Authorization: Bearer $TOKEN" \
     "http://localhost:9000/job-contract/training-config?jobId=credit-risk-v1"
```

Admin example to create a config:

```bash
curl -X POST http://localhost:9000/job-contract/training-config \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "jobId": "credit-risk-v1",
       "modelName": "credit-risk",
       "modelVersion": "1.0.0",
       "datasetUri": "s3://datasets/credit-risk-v3",
       "objective": "Binary classification",
       "roundDurationSec": 300,
       "batchSize": 64,
       "learningRate": 0.05,
       "maxClusterRounds": 10,
       "maxStateRounds": 5,
       "alpha": 0.8
     }'
```
