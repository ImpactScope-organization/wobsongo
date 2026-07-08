package queue

import "github.com/google/uuid"

// ParseDocumentDTO is the river job kind for parsing an ingested document via Docling.
type ParseDocumentDTO struct {
	DocumentID uuid.UUID `json:"document_id"`
	FileKey    string    `json:"file_key"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (ParseDocumentDTO) Kind() string {
	return string(JobTypeParseDocument)
}
