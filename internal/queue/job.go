// Package queue defines interfaces and types for background job processing.
package queue

type BackgroundJobType string

const (
	// JobTypeExtractMedia represents a job type for extracting media from a given source.
	JobTypeExtractMedia BackgroundJobType = "extract_media"

	// JobTypeTranscribeVideo represents a job type specifically for sending audio to Modal.
	JobTypeTranscribeVideo BackgroundJobType = "transcribe_video"
)

// BackgroundJob represents a generic background job.
// It is strictly compatible with River's job system.
type BackgroundJob interface {
	// Kind returns the kind of the job.
	Kind() string
}
