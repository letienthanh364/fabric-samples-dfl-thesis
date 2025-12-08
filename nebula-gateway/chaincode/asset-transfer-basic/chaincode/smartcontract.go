package chaincode

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

// Asset describes basic details of what makes up a simple asset
// Insert struct field in alphabetic order => to achieve determinism across languages
// golang keeps the order when marshal to json but doesn't order automatically
type Asset struct {
	AppraisedValue int    `json:"AppraisedValue"`
	Color          string `json:"Color"`
	ID             string `json:"ID"`
	Owner          string `json:"Owner"`
	Size           int    `json:"Size"`
}

const (
	genesisModelCIDPrefix     = "job-contract:genesis-cid:"
	genesisModelHashPrefix    = "job-contract:genesis-hash:"
	trainingConfigStatePrefix = "job-contract:training-config:"
)

// GenesisModelCID keeps the metadata that points to the canonical genesis model artifact.
type GenesisModelCID struct {
	JobID           string `json:"jobId"`
	CID             string `json:"cid"`
	Purpose         string `json:"purpose"`
	ModelFamily     string `json:"modelFamily"`
	DatasetSummary  string `json:"datasetSummary,omitempty"`
	Notes           string `json:"notes,omitempty"`
	LastUpdatedTime string `json:"updatedAt"`
}

// GenesisModelHash captures the digest used to validate the genesis model contents.
type GenesisModelHash struct {
	JobID           string `json:"jobId"`
	Hash            string `json:"hash"`
	HashAlgorithm   string `json:"hashAlgorithm"`
	ModelFormat     string `json:"modelFormat"`
	Compression     string `json:"compression,omitempty"`
	Notes           string `json:"notes,omitempty"`
	LastUpdatedTime string `json:"updatedAt"`
}

// TrainingConfig captures how a federated job should be executed.
type TrainingConfig struct {
	JobID            string  `json:"jobId"`
	ModelName        string  `json:"modelName"`
	ModelVersion     string  `json:"modelVersion,omitempty"`
	DatasetURI       string  `json:"datasetUri"`
	Objective        string  `json:"objective"`
	Description      string  `json:"description,omitempty"`
	RoundDurationSec int64   `json:"roundDurationSec"`
	BatchSize        int64   `json:"batchSize"`
	LearningRate     float64 `json:"learningRate"`
	MaxClusterRounds int64   `json:"maxClusterRounds"`
	MaxStateRounds   int64   `json:"maxStateRounds"`
	Alpha            float64 `json:"alpha"`
	LastUpdatedTime  string  `json:"updatedAt"`
}

// InitLedger adds a base set of assets to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	assets := []Asset{
		{ID: "asset1", Color: "blue", Size: 5, Owner: "Tomoko", AppraisedValue: 300},
		{ID: "asset2", Color: "red", Size: 5, Owner: "Brad", AppraisedValue: 400},
		{ID: "asset3", Color: "green", Size: 10, Owner: "Jin Soo", AppraisedValue: 500},
		{ID: "asset4", Color: "yellow", Size: 10, Owner: "Max", AppraisedValue: 600},
		{ID: "asset5", Color: "black", Size: 15, Owner: "Adriana", AppraisedValue: 700},
		{ID: "asset6", Color: "white", Size: 15, Owner: "Michel", AppraisedValue: 800},
	}

	for _, asset := range assets {
		assetJSON, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(asset.ID, assetJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}

	return nil
}

// CreateAsset issues a new asset to the world state with given details.
func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, id string, color string, size int, owner string, appraisedValue int) error {
	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the asset %s already exists", id)
	}

	asset := Asset{
		ID:             id,
		Color:          color,
		Size:           size,
		Owner:          owner,
		AppraisedValue: appraisedValue,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// ReadAsset returns the asset stored in the world state with given id.
func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, id string) (*Asset, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("the asset %s does not exist", id)
	}

	var asset Asset
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// UpdateAsset updates an existing asset in the world state with provided parameters.
func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, id string, color string, size int, owner string, appraisedValue int) error {
	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the asset %s does not exist", id)
	}

	// overwriting original asset with new asset
	asset := Asset{
		ID:             id,
		Color:          color,
		Size:           size,
		Owner:          owner,
		AppraisedValue: appraisedValue,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// DeleteAsset deletes an given asset from the world state.
func (s *SmartContract) DeleteAsset(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the asset %s does not exist", id)
	}

	return ctx.GetStub().DelState(id)
}

// AssetExists returns true when asset with given ID exists in world state
func (s *SmartContract) AssetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return assetJSON != nil, nil
}

// TransferAsset updates the owner field of asset with given id in world state, and returns the old owner.
func (s *SmartContract) TransferAsset(ctx contractapi.TransactionContextInterface, id string, newOwner string) (string, error) {
	asset, err := s.ReadAsset(ctx, id)
	if err != nil {
		return "", err
	}

	oldOwner := asset.Owner
	asset.Owner = newOwner

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return "", err
	}

	err = ctx.GetStub().PutState(id, assetJSON)
	if err != nil {
		return "", err
	}

	return oldOwner, nil
}

// GetAllAssets returns all assets found in world state
func (s *SmartContract) GetAllAssets(ctx contractapi.TransactionContextInterface) ([]*Asset, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all assets in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var assets []*Asset
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset Asset
		err = json.Unmarshal(queryResponse.Value, &asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

// UpsertGenesisModelCID stores or updates the content identifier of the genesis model for a job contract.
func (s *SmartContract) UpsertGenesisModelCID(ctx contractapi.TransactionContextInterface, jobID, cid, purpose, modelFamily, datasetSummary, notes string) error {
	if jobID == "" {
		return fmt.Errorf("jobId is required")
	}
	if cid == "" {
		return fmt.Errorf("cid is required")
	}
	if purpose == "" {
		return fmt.Errorf("purpose is required")
	}
	if modelFamily == "" {
		return fmt.Errorf("modelFamily is required")
	}

	timestamp, err := txTimeRFC3339(ctx)
	if err != nil {
		return err
	}

	record := GenesisModelCID{
		JobID:           jobID,
		CID:             cid,
		Purpose:         purpose,
		ModelFamily:     modelFamily,
		DatasetSummary:  datasetSummary,
		Notes:           notes,
		LastUpdatedTime: timestamp,
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(genesisModelCIDKey(jobID), payload)
}

// GetGenesisModelCID returns the stored genesis model CID metadata for a job contract.
func (s *SmartContract) GetGenesisModelCID(ctx contractapi.TransactionContextInterface, jobID string) (*GenesisModelCID, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobId is required")
	}
	payload, err := ctx.GetStub().GetState(genesisModelCIDKey(jobID))
	if err != nil {
		return nil, fmt.Errorf("failed to read genesis model cid: %w", err)
	}
	if payload == nil {
		return nil, fmt.Errorf("genesis model cid for %s does not exist", jobID)
	}
	var record GenesisModelCID
	if err := json.Unmarshal(payload, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// UpsertGenesisModelHash stores or updates the integrity metadata for a genesis model.
func (s *SmartContract) UpsertGenesisModelHash(ctx contractapi.TransactionContextInterface, jobID, hash, hashAlgorithm, modelFormat, compression, notes string) error {
	if jobID == "" {
		return fmt.Errorf("jobId is required")
	}
	if hash == "" {
		return fmt.Errorf("hash is required")
	}
	if hashAlgorithm == "" {
		return fmt.Errorf("hashAlgorithm is required")
	}
	if modelFormat == "" {
		return fmt.Errorf("modelFormat is required")
	}

	timestamp, err := txTimeRFC3339(ctx)
	if err != nil {
		return err
	}

	record := GenesisModelHash{
		JobID:           jobID,
		Hash:            hash,
		HashAlgorithm:   hashAlgorithm,
		ModelFormat:     modelFormat,
		Compression:     compression,
		Notes:           notes,
		LastUpdatedTime: timestamp,
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(genesisModelHashKey(jobID), payload)
}

// GetGenesisModelHash returns the stored hash metadata for a job contract genesis model.
func (s *SmartContract) GetGenesisModelHash(ctx contractapi.TransactionContextInterface, jobID string) (*GenesisModelHash, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobId is required")
	}
	payload, err := ctx.GetStub().GetState(genesisModelHashKey(jobID))
	if err != nil {
		return nil, fmt.Errorf("failed to read genesis model hash: %w", err)
	}
	if payload == nil {
		return nil, fmt.Errorf("genesis model hash for %s does not exist", jobID)
	}
	var record GenesisModelHash
	if err := json.Unmarshal(payload, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// UpsertTrainingConfig stores or updates the training plan for a job.
func (s *SmartContract) UpsertTrainingConfig(ctx contractapi.TransactionContextInterface, jobID, modelName, modelVersion, datasetURI, objective, description, roundDurationSecStr, batchSizeStr, learningRateStr, maxClusterRoundsStr, maxStateRoundsStr, alphaStr string) error {
	if jobID == "" {
		return fmt.Errorf("jobId is required")
	}
	if modelName == "" {
		return fmt.Errorf("modelName is required")
	}
	if datasetURI == "" {
		return fmt.Errorf("datasetUri is required")
	}
	if objective == "" {
		return fmt.Errorf("objective is required")
	}

	roundDurationSec, err := parsePositiveInt(roundDurationSecStr, "roundDurationSec")
	if err != nil {
		return err
	}
	batchSize, err := parsePositiveInt(batchSizeStr, "batchSize")
	if err != nil {
		return err
	}
	maxClusterRounds, err := parsePositiveInt(maxClusterRoundsStr, "maxClusterRounds")
	if err != nil {
		return err
	}
	maxStateRounds, err := parsePositiveInt(maxStateRoundsStr, "maxStateRounds")
	if err != nil {
		return err
	}
	learningRate, err := parsePositiveFloat(learningRateStr, "learningRate")
	if err != nil {
		return err
	}
	alpha, err := parsePositiveFloat(alphaStr, "alpha")
	if err != nil {
		return err
	}

	timestamp, err := txTimeRFC3339(ctx)
	if err != nil {
		return err
	}

	record := TrainingConfig{
		JobID:            jobID,
		ModelName:        modelName,
		ModelVersion:     modelVersion,
		DatasetURI:       datasetURI,
		Objective:        objective,
		Description:      description,
		RoundDurationSec: roundDurationSec,
		BatchSize:        batchSize,
		LearningRate:     learningRate,
		MaxClusterRounds: maxClusterRounds,
		MaxStateRounds:   maxStateRounds,
		Alpha:            alpha,
		LastUpdatedTime:  timestamp,
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(trainingConfigKey(jobID), payload)
}

// GetTrainingConfig returns the stored training config for a jobID.
func (s *SmartContract) GetTrainingConfig(ctx contractapi.TransactionContextInterface, jobID string) (*TrainingConfig, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobId is required")
	}
	payload, err := ctx.GetStub().GetState(trainingConfigKey(jobID))
	if err != nil {
		return nil, fmt.Errorf("failed to read training config: %w", err)
	}
	if payload == nil {
		return nil, fmt.Errorf("training config for %s does not exist", jobID)
	}
	var record TrainingConfig
	if err := json.Unmarshal(payload, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func genesisModelCIDKey(jobID string) string {
	return genesisModelCIDPrefix + jobID
}

func genesisModelHashKey(jobID string) string {
	return genesisModelHashPrefix + jobID
}

func trainingConfigKey(jobID string) string {
	return trainingConfigStatePrefix + jobID
}

func txTimeRFC3339(ctx contractapi.TransactionContextInterface) (string, error) {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return "", fmt.Errorf("failed to fetch transaction timestamp: %w", err)
	}
	seconds := ts.GetSeconds()
	nanos := ts.GetNanos()
	return time.Unix(seconds, int64(nanos)).UTC().Format(time.RFC3339Nano), nil
}

func parsePositiveInt(value, field string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a positive integer: %w", field, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", field)
	}
	return parsed, nil
}

func parsePositiveFloat(value, field string) (float64, error) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a positive number: %w", field, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", field)
	}
	return parsed, nil
}
