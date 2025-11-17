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
