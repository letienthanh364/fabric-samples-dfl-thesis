package model

import "errors"

// GenesisModelCIDRequest describes the payload needed to track the genesis model CID.
type GenesisModelCIDRequest struct {
	JobID          string `json:"jobId"`
	CID            string `json:"cid"`
	Purpose        string `json:"purpose"`
	ModelFamily    string `json:"modelFamily"`
	DatasetSummary string `json:"datasetSummary"`
	Notes          string `json:"notes"`
}

// GenesisModelCIDRecord represents the ledger state for a genesis model CID.
type GenesisModelCIDRecord struct {
	JobID          string `json:"jobId"`
	CID            string `json:"cid"`
	Purpose        string `json:"purpose"`
	ModelFamily    string `json:"modelFamily"`
	DatasetSummary string `json:"datasetSummary,omitempty"`
	Notes          string `json:"notes,omitempty"`
	UpdatedAt      string `json:"updatedAt"`
}

// GenesisModelHashRequest describes the payload needed to register a model hash.
type GenesisModelHashRequest struct {
	JobID         string `json:"jobId"`
	Hash          string `json:"hash"`
	HashAlgorithm string `json:"hashAlgorithm"`
	ModelFormat   string `json:"modelFormat"`
	Compression   string `json:"compression"`
	Notes         string `json:"notes"`
}

// GenesisModelHashRecord mirrors the ledger entry for a hash.
type GenesisModelHashRecord struct {
	JobID         string `json:"jobId"`
	Hash          string `json:"hash"`
	HashAlgorithm string `json:"hashAlgorithm"`
	ModelFormat   string `json:"modelFormat"`
	Compression   string `json:"compression,omitempty"`
	Notes         string `json:"notes,omitempty"`
	UpdatedAt     string `json:"updatedAt"`
}

// TrainingConfigRequest captures the parameters needed to configure a job's DFL run.
type TrainingConfigRequest struct {
	JobID            string  `json:"jobId"`
	ModelName        string  `json:"modelName"`
	ModelVersion     string  `json:"modelVersion"`
	DatasetURI       string  `json:"datasetUri"`
	Objective        string  `json:"objective"`
	Description      string  `json:"description"`
	RoundDurationSec int64   `json:"roundDurationSec"`
	BatchSize        int64   `json:"batchSize"`
	LearningRate     float64 `json:"learningRate"`
	MaxClusterRounds int64   `json:"maxClusterRounds"`
	MaxStateRounds   int64   `json:"maxStateRounds"`
	Alpha            float64 `json:"alpha"`
}

// TrainingConfigRecord mirrors the configuration stored on-chain.
type TrainingConfigRecord struct {
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
	UpdatedAt        string  `json:"updatedAt"`
}

func (r GenesisModelCIDRequest) Validate() error {
	switch {
	case r.JobID == "":
		return errors.New("jobId is required")
	case r.CID == "":
		return errors.New("cid is required")
	case r.Purpose == "":
		return errors.New("purpose is required")
	case r.ModelFamily == "":
		return errors.New("modelFamily is required")
	default:
		return nil
	}
}

func (r GenesisModelHashRequest) Validate() error {
	switch {
	case r.JobID == "":
		return errors.New("jobId is required")
	case r.Hash == "":
		return errors.New("hash is required")
	case r.HashAlgorithm == "":
		return errors.New("hashAlgorithm is required")
	case r.ModelFormat == "":
		return errors.New("modelFormat is required")
	default:
		return nil
	}
}

func (r TrainingConfigRequest) Validate() error {
	switch {
	case r.JobID == "":
		return errors.New("jobId is required")
	case r.ModelName == "":
		return errors.New("modelName is required")
	case r.DatasetURI == "":
		return errors.New("datasetUri is required")
	case r.Objective == "":
		return errors.New("objective is required")
	case r.RoundDurationSec <= 0:
		return errors.New("roundDurationSec must be greater than zero")
	case r.BatchSize <= 0:
		return errors.New("batchSize must be greater than zero")
	case r.LearningRate <= 0:
		return errors.New("learningRate must be greater than zero")
	case r.MaxClusterRounds <= 0:
		return errors.New("maxClusterRounds must be greater than zero")
	case r.MaxStateRounds <= 0:
		return errors.New("maxStateRounds must be greater than zero")
	case r.Alpha <= 0:
		return errors.New("alpha must be greater than zero")
	default:
		return nil
	}
}
