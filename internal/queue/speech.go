package queue

import "github.com/riverqueue/river"

// speechJobMaxAttempts is how many times River runs a SpeechJob before giving up.
const speechJobMaxAttempts = 2

// SpeechJob turns one bot reply into a voice message: the text is cleaned,
// translated to English when Language is "en", synthesized, and the resulting
// audio is delivered to wa-bot through BotClient.NotifyTTSDone.
type SpeechJob struct {
	// JobID is the correlation ID wa-bot waits on for the audio callback.
	JobID string `json:"job_id"`

	// Text is the bot's (French) reply to speak.
	Text string `json:"text"`

	// Language is the language the audio is spoken in: "fr" or "en".
	Language string `json:"language"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (SpeechJob) Kind() string { return string(JobTypeSynthesizeSpeech) }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SpeechJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueSpeechSynthesis,
		MaxAttempts: speechJobMaxAttempts,
	}
}
