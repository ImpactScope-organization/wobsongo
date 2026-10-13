package queue

import (
	"time"

	"github.com/riverqueue/river"
)

const (
	// speechJobMaxAttempts is how many times River runs a SpeechJob. A single
	// attempt keeps the worst-case wait deterministic: wa-bot sends the text
	// answer when the audio takes longer than its own wait, so a late retry
	// would only produce audio nobody uses.
	speechJobMaxAttempts = 1

	// DefaultSpeechMaxWorkers is the default number of speech jobs one server
	// process synthesizes at the same time.
	DefaultSpeechMaxWorkers = 4

	// DefaultSpeechMaxPendingJobs is the default maximum number of SpeechJobs
	// allowed to wait in the queue
	DefaultSpeechMaxPendingJobs = 50

	// SpeechJobTTL is how long a speech job stays useful. A job older than this
	// is dropped without synthesizing, and a running one is canceled at this age.
	SpeechJobTTL = 4 * time.Minute

	// SpeechNotifyTimeout bounds the callback that delivers the result to wa-bot.
	SpeechNotifyTimeout = 30 * time.Second
)

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
