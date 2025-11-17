package chaincode_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/queryresult"
	"github.com/hyperledger/fabric-samples/asset-transfer-basic/chaincode-go/chaincode"
	"github.com/hyperledger/fabric-samples/asset-transfer-basic/chaincode-go/chaincode/mocks"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate counterfeiter -o mocks/transaction.go -fake-name TransactionContext . transactionContext
type transactionContext interface {
	contractapi.TransactionContextInterface
}

//go:generate counterfeiter -o mocks/chaincodestub.go -fake-name ChaincodeStub . chaincodeStub
type chaincodeStub interface {
	shim.ChaincodeStubInterface
}

//go:generate counterfeiter -o mocks/statequeryiterator.go -fake-name StateQueryIterator . stateQueryIterator
type stateQueryIterator interface {
	shim.StateQueryIteratorInterface
}

func TestInitLedger(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	assetTransfer := chaincode.SmartContract{}
	err := assetTransfer.InitLedger(transactionContext)
	require.NoError(t, err)

	chaincodeStub.PutStateReturns(fmt.Errorf("failed inserting key"))
	err = assetTransfer.InitLedger(transactionContext)
	require.EqualError(t, err, "failed to put to world state. failed inserting key")
}

func TestCreateAsset(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	assetTransfer := chaincode.SmartContract{}
	err := assetTransfer.CreateAsset(transactionContext, "", "", 0, "", 0)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns([]byte{}, nil)
	err = assetTransfer.CreateAsset(transactionContext, "asset1", "", 0, "", 0)
	require.EqualError(t, err, "the asset asset1 already exists")

	chaincodeStub.GetStateReturns(nil, fmt.Errorf("unable to retrieve asset"))
	err = assetTransfer.CreateAsset(transactionContext, "asset1", "", 0, "", 0)
	require.EqualError(t, err, "failed to read from world state: unable to retrieve asset")
}

func TestReadAsset(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	expectedAsset := &chaincode.Asset{ID: "asset1"}
	bytes, err := json.Marshal(expectedAsset)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	assetTransfer := chaincode.SmartContract{}
	asset, err := assetTransfer.ReadAsset(transactionContext, "")
	require.NoError(t, err)
	require.Equal(t, expectedAsset, asset)

	chaincodeStub.GetStateReturns(nil, fmt.Errorf("unable to retrieve asset"))
	_, err = assetTransfer.ReadAsset(transactionContext, "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve asset")

	chaincodeStub.GetStateReturns(nil, nil)
	asset, err = assetTransfer.ReadAsset(transactionContext, "asset1")
	require.EqualError(t, err, "the asset asset1 does not exist")
	require.Nil(t, asset)
}

func TestUpdateAsset(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	expectedAsset := &chaincode.Asset{ID: "asset1"}
	bytes, err := json.Marshal(expectedAsset)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	assetTransfer := chaincode.SmartContract{}
	err = assetTransfer.UpdateAsset(transactionContext, "", "", 0, "", 0)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(nil, nil)
	err = assetTransfer.UpdateAsset(transactionContext, "asset1", "", 0, "", 0)
	require.EqualError(t, err, "the asset asset1 does not exist")

	chaincodeStub.GetStateReturns(nil, fmt.Errorf("unable to retrieve asset"))
	err = assetTransfer.UpdateAsset(transactionContext, "asset1", "", 0, "", 0)
	require.EqualError(t, err, "failed to read from world state: unable to retrieve asset")
}

func TestDeleteAsset(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	asset := &chaincode.Asset{ID: "asset1"}
	bytes, err := json.Marshal(asset)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	chaincodeStub.DelStateReturns(nil)
	assetTransfer := chaincode.SmartContract{}
	err = assetTransfer.DeleteAsset(transactionContext, "")
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(nil, nil)
	err = assetTransfer.DeleteAsset(transactionContext, "asset1")
	require.EqualError(t, err, "the asset asset1 does not exist")

	chaincodeStub.GetStateReturns(nil, fmt.Errorf("unable to retrieve asset"))
	err = assetTransfer.DeleteAsset(transactionContext, "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve asset")
}

func TestTransferAsset(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	asset := &chaincode.Asset{ID: "asset1"}
	bytes, err := json.Marshal(asset)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	assetTransfer := chaincode.SmartContract{}
	_, err = assetTransfer.TransferAsset(transactionContext, "", "")
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(nil, fmt.Errorf("unable to retrieve asset"))
	_, err = assetTransfer.TransferAsset(transactionContext, "", "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve asset")
}

func TestGetAllAssets(t *testing.T) {
	asset := &chaincode.Asset{ID: "asset1"}
	bytes, err := json.Marshal(asset)
	require.NoError(t, err)

	iterator := &mocks.StateQueryIterator{}
	iterator.HasNextReturnsOnCall(0, true)
	iterator.HasNextReturnsOnCall(1, false)
	iterator.NextReturns(&queryresult.KV{Value: bytes}, nil)

	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	chaincodeStub.GetStateByRangeReturns(iterator, nil)
	assetTransfer := &chaincode.SmartContract{}
	assets, err := assetTransfer.GetAllAssets(transactionContext)
	require.NoError(t, err)
	require.Equal(t, []*chaincode.Asset{asset}, assets)

	iterator.HasNextReturns(true)
	iterator.NextReturns(nil, fmt.Errorf("failed retrieving next item"))
	assets, err = assetTransfer.GetAllAssets(transactionContext)
	require.EqualError(t, err, "failed retrieving next item")
	require.Nil(t, assets)

	chaincodeStub.GetStateByRangeReturns(nil, fmt.Errorf("failed retrieving all assets"))
	assets, err = assetTransfer.GetAllAssets(transactionContext)
	require.EqualError(t, err, "failed retrieving all assets")
	require.Nil(t, assets)
}

func TestUpsertGenesisModelCID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		chaincodeStub := &mocks.ChaincodeStub{}
		transactionContext := &mocks.TransactionContext{}
		transactionContext.GetStubReturns(chaincodeStub)
		chaincodeStub.GetTxTimestampReturns(timestamppb.New(time.Unix(1700000000, 0)), nil)

		assetTransfer := chaincode.SmartContract{}
		err := assetTransfer.UpsertGenesisModelCID(transactionContext, "job1", "cid123", "fraud detection", "cnn", "tabular finance", "seed weights")
		require.NoError(t, err)
		require.Equal(t, 1, chaincodeStub.PutStateCallCount())

		key, payload := chaincodeStub.PutStateArgsForCall(0)
		require.Equal(t, "job-contract:genesis-cid:job1", key)
		var stored chaincode.GenesisModelCID
		require.NoError(t, json.Unmarshal(payload, &stored))
		require.Equal(t, "cid123", stored.CID)
		require.Equal(t, "fraud detection", stored.Purpose)
	})

	t.Run("validation errors", func(t *testing.T) {
		assetTransfer := chaincode.SmartContract{}
		err := assetTransfer.UpsertGenesisModelCID(&mocks.TransactionContext{}, "", "", "", "", "", "")
		require.EqualError(t, err, "jobId is required")
	})

	t.Run("timestamp failure", func(t *testing.T) {
		chaincodeStub := &mocks.ChaincodeStub{}
		transactionContext := &mocks.TransactionContext{}
		transactionContext.GetStubReturns(chaincodeStub)
		chaincodeStub.GetTxTimestampReturns(nil, fmt.Errorf("ts error"))

		assetTransfer := chaincode.SmartContract{}
		err := assetTransfer.UpsertGenesisModelCID(transactionContext, "job1", "cid", "purpose", "cnn", "", "")
		require.EqualError(t, err, "failed to fetch transaction timestamp: ts error")
	})

	t.Run("put state failure", func(t *testing.T) {
		chaincodeStub := &mocks.ChaincodeStub{}
		transactionContext := &mocks.TransactionContext{}
		transactionContext.GetStubReturns(chaincodeStub)
		chaincodeStub.GetTxTimestampReturns(timestamppb.New(time.Unix(1700000000, 0)), nil)
		chaincodeStub.PutStateReturns(fmt.Errorf("put failure"))

		assetTransfer := chaincode.SmartContract{}
		err := assetTransfer.UpsertGenesisModelCID(transactionContext, "job1", "cid", "purpose", "cnn", "", "")
		require.EqualError(t, err, "put failure")
	})
}

func TestGetGenesisModelCID(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	record := &chaincode.GenesisModelCID{
		JobID:       "job1",
		CID:         "cid321",
		Purpose:     "imagery",
		ModelFamily: "vit",
	}
	payload, err := json.Marshal(record)
	require.NoError(t, err)
	chaincodeStub.GetStateReturns(payload, nil)

	assetTransfer := chaincode.SmartContract{}
	result, err := assetTransfer.GetGenesisModelCID(transactionContext, "job1")
	require.NoError(t, err)
	require.Equal(t, record, result)

	_, err = assetTransfer.GetGenesisModelCID(transactionContext, "")
	require.EqualError(t, err, "jobId is required")

	chaincodeStub.GetStateReturns(nil, fmt.Errorf("boom"))
	_, err = assetTransfer.GetGenesisModelCID(transactionContext, "job1")
	require.EqualError(t, err, "failed to read genesis model cid: boom")

	chaincodeStub.GetStateReturns(nil, nil)
	_, err = assetTransfer.GetGenesisModelCID(transactionContext, "job1")
	require.EqualError(t, err, "genesis model cid for job1 does not exist")
}

func TestUpsertGenesisModelHash(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		chaincodeStub := &mocks.ChaincodeStub{}
		transactionContext := &mocks.TransactionContext{}
		transactionContext.GetStubReturns(chaincodeStub)
		chaincodeStub.GetTxTimestampReturns(timestamppb.New(time.Unix(1700000000, 0)), nil)

		assetTransfer := chaincode.SmartContract{}
		err := assetTransfer.UpsertGenesisModelHash(transactionContext, "job1", "abc123", "sha256", "onnx", "gzip", "hash of the initial weights")
		require.NoError(t, err)
		require.Equal(t, 1, chaincodeStub.PutStateCallCount())

		key, payload := chaincodeStub.PutStateArgsForCall(0)
		require.Equal(t, "job-contract:genesis-hash:job1", key)
		var stored chaincode.GenesisModelHash
		require.NoError(t, json.Unmarshal(payload, &stored))
		require.Equal(t, "sha256", stored.HashAlgorithm)
		require.Equal(t, "onnx", stored.ModelFormat)
	})

	t.Run("validation errors", func(t *testing.T) {
		assetTransfer := chaincode.SmartContract{}
		err := assetTransfer.UpsertGenesisModelHash(&mocks.TransactionContext{}, "", "", "", "", "", "")
		require.EqualError(t, err, "jobId is required")
	})

	t.Run("timestamp failure", func(t *testing.T) {
		chaincodeStub := &mocks.ChaincodeStub{}
		transactionContext := &mocks.TransactionContext{}
		transactionContext.GetStubReturns(chaincodeStub)
		chaincodeStub.GetTxTimestampReturns(nil, fmt.Errorf("ts error"))

		assetTransfer := chaincode.SmartContract{}
		err := assetTransfer.UpsertGenesisModelHash(transactionContext, "job1", "abc", "sha256", "pth", "", "")
		require.EqualError(t, err, "failed to fetch transaction timestamp: ts error")
	})

	t.Run("put state failure", func(t *testing.T) {
		chaincodeStub := &mocks.ChaincodeStub{}
		transactionContext := &mocks.TransactionContext{}
		transactionContext.GetStubReturns(chaincodeStub)
		chaincodeStub.GetTxTimestampReturns(timestamppb.New(time.Unix(1700000000, 0)), nil)
		chaincodeStub.PutStateReturns(fmt.Errorf("put failure"))

		assetTransfer := chaincode.SmartContract{}
		err := assetTransfer.UpsertGenesisModelHash(transactionContext, "job1", "abc", "sha256", "pth", "", "")
		require.EqualError(t, err, "put failure")
	})
}

func TestGetGenesisModelHash(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	record := &chaincode.GenesisModelHash{
		JobID:         "job1",
		Hash:          "deadbeef",
		HashAlgorithm: "sha256",
		ModelFormat:   "h5",
	}
	payload, err := json.Marshal(record)
	require.NoError(t, err)
	chaincodeStub.GetStateReturns(payload, nil)

	assetTransfer := chaincode.SmartContract{}
	result, err := assetTransfer.GetGenesisModelHash(transactionContext, "job1")
	require.NoError(t, err)
	require.Equal(t, record, result)

	_, err = assetTransfer.GetGenesisModelHash(transactionContext, "")
	require.EqualError(t, err, "jobId is required")

	chaincodeStub.GetStateReturns(nil, fmt.Errorf("boom"))
	_, err = assetTransfer.GetGenesisModelHash(transactionContext, "job1")
	require.EqualError(t, err, "failed to read genesis model hash: boom")

	chaincodeStub.GetStateReturns(nil, nil)
	_, err = assetTransfer.GetGenesisModelHash(transactionContext, "job1")
	require.EqualError(t, err, "genesis model hash for job1 does not exist")
}
