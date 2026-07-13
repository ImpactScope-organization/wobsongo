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

// EmbedChunksDTO is the river job kind for computing and storing embeddings
// for all of a document's chunks that have text but no embedding yet.
type EmbedChunksDTO struct {
	DocumentID uuid.UUID `json:"document_id"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (EmbedChunksDTO) Kind() string {
	return string(JobTypeEmbedChunks)
}

// ExtractKnowledgeDTO is the river job kind for extracting atomic knowledge
// facts from all of a document's chunks that have text but haven't had
// extraction run yet.
type ExtractKnowledgeDTO struct {
	DocumentID uuid.UUID `json:"document_id"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (ExtractKnowledgeDTO) Kind() string {
	return string(JobTypeExtractKnowledge)
}
