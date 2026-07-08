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
)

// BackgroundJob represents a generic background job.
// It is strictly compatible with River's job system.
type BackgroundJob interface {
	// Kind returns the kind of the job.
	Kind() string
}
