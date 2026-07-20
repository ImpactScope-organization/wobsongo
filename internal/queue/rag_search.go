package queue

import "github.com/riverqueue/river"

// RAGSearchJob carries the data required to perform a RAG search.
type RAGSearchJob struct {
	ExtractionID string `json:"extractionId"`
	Transcript   string `json:"transcript"`
}

// Kind returns the River job type for RAG search.
func (RAGSearchJob) Kind() string { return string(JobTypeRagSearch) }

// InsertOpts routes this job to the media-processing queue.
func (RAGSearchJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueMediaProcessing,
	}
}