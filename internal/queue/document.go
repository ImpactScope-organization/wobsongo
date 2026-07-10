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

// ProcessParsedDocumentDTO is the river job kind for turning a document's
// raw Docling output (already stored in S3 by ParseDocumentWorker) into
// stored chunks.
type ProcessParsedDocumentDTO struct {
	DocumentID   uuid.UUID `json:"document_id"`
	RawOutputKey string    `json:"raw_output_key"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (ProcessParsedDocumentDTO) Kind() string {
	return string(JobTypeProcessParsedDocument)
}

// CaptionImageChunksDTO is the river job kind for generating and storing
// captions for a set of image/chart chunks belonging to one document.
type CaptionImageChunksDTO struct {
	DocumentID uuid.UUID   `json:"document_id"`
	ChunkIDs   []uuid.UUID `json:"chunk_ids"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (CaptionImageChunksDTO) Kind() string {
	return string(JobTypeCaptionImageChunks)
}
