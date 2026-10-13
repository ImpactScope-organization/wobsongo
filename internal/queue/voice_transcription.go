package queue

import "github.com/riverqueue/river"

// voiceTranscriptionMaxAttempts is 1 because the user is waiting for an
// answer; a late retry would only produce a reply after the user gave up.
const voiceTranscriptionMaxAttempts = 1

// VoiceTranscriptionJob transcribes one inbound voice message and then starts
// an agent turn with the transcript as the user's message.
type VoiceTranscriptionJob struct {
	// ExtractionID is the correlation ID wa-bot waits on for the answer.
	ExtractionID string `json:"extraction_id"`

	// Jid is the chat identifier the voice message came from.
	Jid string `json:"jid"`

	// Format is the audio format hint.
	Format string `json:"format"`

	// Audio is the raw audio (encoded as base64 by encoding/json).
	Audio []byte `json:"audio"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (VoiceTranscriptionJob) Kind() string { return string(JobTypeTranscribeVoice) }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (VoiceTranscriptionJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueMediaProcessing,
		MaxAttempts: voiceTranscriptionMaxAttempts,
	}
}
