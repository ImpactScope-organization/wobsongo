package queue

import (
	"time"

	"github.com/riverqueue/river"
)

const (
	// TTSAudioRetention is how long a synthesized reply stays in S3. The
	// presigned URL handed to Twilio/Telegram lives 15 minutes, so a day is
	// far more than enough.
	TTSAudioRetention = 24 * time.Hour

	// TTSAudioCleanupInterval is how often the cleanup job runs.
	TTSAudioCleanupInterval = time.Hour

	// MaintenanceMaxWorkers is the number of housekeeping jobs run at once.
	MaintenanceMaxWorkers = 1
)

// TTSAudioCleanupJob deletes synthesized voice replies older than
// TTSAudioRetention from S3. It is enqueued periodically by River.
type TTSAudioCleanupJob struct{}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (TTSAudioCleanupJob) Kind() string { return string(JobTypeCleanupTTSAudio) }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (TTSAudioCleanupJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueMaintenance,
		// The next periodic run retries anyway.
		MaxAttempts: 1,
	}
}
