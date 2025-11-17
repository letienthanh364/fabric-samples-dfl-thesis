package model

import "errors"

// Asset encapsulates the ledger representation of an asset.
type Asset struct {
	ID             string `json:"id"`
	Color          string `json:"color"`
	Size           int    `json:"size"`
	Owner          string `json:"owner"`
	AppraisedValue int    `json:"appraisedValue"`
}

func (a Asset) Validate() error {
	switch {
	case a.ID == "", a.Owner == "":
		return errors.New("id and owner are required")
	case a.Size <= 0:
		return errors.New("size must be positive")
	case a.AppraisedValue < 0:
		return errors.New("appraisedValue must be non-negative")
	default:
		return nil
	}
}
