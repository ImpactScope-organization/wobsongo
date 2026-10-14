package queue

import "github.com/riverqueue/river"

// voiceTranscriptionMaxAttempts is 1 because the user is waiting for an
// answer; a late retry would only produce a reply after the user gave up.
const voiceTranscriptionMaxAttempts = 1

// VoiceTranscriptionJob transcribes one inbound voice message and then starts
// an agent turn with the transcript as the user's message. It carries only a
// reference to the media, never the audio: the worker downloads it, stores it
// in S3 and lets the ASR service fetch it by presigned URL.
type VoiceTranscriptionJob struct {
	// ExtractionID is the correlation ID wa-bot waits on for the answer.
	ExtractionID string `json:"extraction_id"`

	// JID is the chat identifier the voice message came from.
	JID string `json:"jid"`

	// Channel is the messaging platform: "whatsapp" or "telegram".
	Channel string `json:"channel"`

	// MediaRef is a Twilio media URL (whatsapp) or a file_id (telegram).
	MediaRef string `json:"media_ref"`

	// MimeType is the audio MIME type reported by the platform.
	MimeType string `json:"mime_type"`
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
