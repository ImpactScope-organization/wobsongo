// Package queue defines interfaces and types for background job processing.
package queue

type BackgroundJobType string

const (
	// JobTypeExtractMedia represents a job type for extracting media from a given source.
	JobTypeExtractMedia BackgroundJobType = "extract_media"

	// JobTypeProcessAITask represents a job type for processing AI tasks.
	JobTypeProcessAITask BackgroundJobType = "process_ai_task"

	// JobTypeParseDocument represents a job type for parsing an ingested document via Docling.
	JobTypeParseDocument BackgroundJobType = "parse_document"

	// JobTypeProcessParsedDocument represents a job type for turning a
	// document's raw Docling output (stored in S3 by ParseDocumentWorker)
	// into stored chunks.
	JobTypeProcessParsedDocument BackgroundJobType = "process_parsed_document"

	// JobTypeCaptionImageChunks represents a job type for generating and
	// storing captions for image/chart chunks extracted during processing.
	JobTypeCaptionImageChunks BackgroundJobType = "caption_image_chunks"

	// JobTypeEmbedChunks represents a job type for computing and storing
	// embeddings for a document's text-bearing, not-yet-embedded chunks.
	JobTypeEmbedChunks BackgroundJobType = "embed_chunks"

	// JobTypeExtractKnowledge represents a job type for extracting atomic
	// knowledge facts from a document's text-bearing, not-yet-extracted chunks.
	JobTypeExtractKnowledge BackgroundJobType = "extract_knowledge"

	// JobTypeEmbedKnowledge represents a job type for computing and storing
	// embeddings for a document's not-yet-embedded atomic knowledge facts.
	JobTypeEmbedKnowledge BackgroundJobType = "embed_knowledge"
)

// BackgroundJob represents a generic background job.
// It is strictly compatible with River's job system.
type BackgroundJob interface {
	// Kind returns the kind of the job.
	Kind() string
}
