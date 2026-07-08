package data

import (
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

// DocumentRepoer defines the interface for interacting with document data storage.
type DocumentRepoer interface {
	CrudableWithTx[model.Document, DocumentRepoer]
	queue.JobEnqueuer
}
