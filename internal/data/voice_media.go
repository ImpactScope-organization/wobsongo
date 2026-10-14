package data

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// MaxVoiceBytes is the largest voice message accepted, both when it is
// downloaded from the messaging platform and when it is stored.
const MaxVoiceBytes int64 = 16 * 1024 * 1024

// VoiceChannel identifies the messaging platform a voice message came from.
type VoiceChannel string

const (
	// VoiceChannelWhatsApp is a WhatsApp message delivered through Twilio.
	VoiceChannelWhatsApp VoiceChannel = "whatsapp"

	// VoiceChannelTelegram is a Telegram message.
	VoiceChannelTelegram VoiceChannel = "telegram"
)

var (
	// ErrVoiceTooLarge is returned when a voice message exceeds MaxVoiceBytes.
	ErrVoiceTooLarge = errors.New("voice message is too large")

	// ErrVoiceEmpty is returned when a downloaded voice message has no bytes.
	ErrVoiceEmpty = errors.New("voice message is empty")

	// ErrInvalidVoiceMediaRef is returned when a media reference is malformed
	// or points at a host we do not download from.
	ErrInvalidVoiceMediaRef = errors.New("invalid voice media reference")
)

// VoiceMediaDownloader fetches an inbound voice message from the messaging
// platform it was sent on.
type VoiceMediaDownloader interface {
	// Download returns the audio bytes behind mediaRef: a Twilio media URL for
	// WhatsApp, or a file_id for Telegram. It returns ErrVoiceTooLarge when the
	// audio exceeds MaxVoiceBytes.
	Download(ctx context.Context, channel VoiceChannel, mediaRef string) ([]byte, error)
}

// VoiceObjectStore is the slice of the S3 provider the voice workers need:
// raw reads/writes plus cleanup of the temporary audio.
type VoiceObjectStore interface {
	RawObjectStore

	// DeleteObject removes the object with the given key.
	DeleteObject(ctx context.Context, key string) error
}

// voiceMimeFormats maps accepted audio MIME types to the format hint the ASR
// service expects. The formats must also be allowed by the S3 filename check
// in the validation package.
var voiceMimeFormats = map[string]string{
	"audio/ogg":   "ogg",
	"audio/opus":  "ogg",
	"audio/mpeg":  "mp3",
	"audio/mp3":   "mp3",
	"audio/mp4":   "m4a",
	"audio/aac":   "aac",
	"audio/amr":   "amr",
	"audio/webm":  "webm",
	"audio/wav":   "wav",
	"audio/x-wav": "wav",
}

// VoiceFormatFromMime returns the audio format hint for mimeType. The boolean
// is false when the MIME type is not an accepted voice format.
func VoiceFormatFromMime(mimeType string) (string, bool) {
	base, _, _ := strings.Cut(mimeType, ";")
	format, ok := voiceMimeFormats[strings.ToLower(strings.TrimSpace(base))]
	return format, ok
}

var telegramFileIDRegex = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)

// ValidateVoiceMediaRef checks that mediaRef is something we may download for
// channel: an https Twilio URL for WhatsApp, a plain file_id for Telegram.
// This is what stops the inbound-voice endpoint being used to fetch arbitrary urls.
func ValidateVoiceMediaRef(channel VoiceChannel, mediaRef string) error {
	switch channel {
	case VoiceChannelWhatsApp:
		u, err := url.Parse(mediaRef)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidVoiceMediaRef, err)
		}
		host := strings.ToLower(u.Hostname())
		trusted := host == "api.twilio.com" || strings.HasSuffix(host, ".twilio.com")
		if u.Scheme != "https" || u.User != nil || !trusted {
			return fmt.Errorf("%w: untrusted media URL host %q", ErrInvalidVoiceMediaRef, host)
		}
		return nil
	case VoiceChannelTelegram:
		if !telegramFileIDRegex.MatchString(mediaRef) {
			return fmt.Errorf("%w: malformed telegram file_id", ErrInvalidVoiceMediaRef)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown channel %q", ErrInvalidVoiceMediaRef, channel)
	}
}
