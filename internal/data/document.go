package data

import "github.com/impactscope-organization/wobsongo/internal/model"

// DocumentRepoer defines the interface for interacting with document data storage.
type DocumentRepoer interface {
	CrudableWithTx[model.Document, DocumentRepoer]
}
