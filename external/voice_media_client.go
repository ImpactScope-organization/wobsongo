package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/data"
)

const (
	voiceDownloadTimeout = 30 * time.Second
	telegramAPIBaseURL   = "https://api.telegram.org"

	// maxTelegramInfoBytes bounds the getFile JSON response.
	maxTelegramInfoBytes = 1 * 1024 * 1024
)

// VoiceMediaClient implements data.VoiceMediaDownloader for Twilio (WhatsApp)
// and Telegram. It holds the platform credentials, so wa-bot never has to.
type VoiceMediaClient struct {
	httpClient       *http.Client
	twilioAccountSID string
	twilioAuthToken  string
	telegramBotToken string
	telegramBaseURL  string
}

// NewVoiceMediaClient creates a VoiceMediaClient.
func NewVoiceMediaClient(
	twilioAccountSID, twilioAuthToken, telegramBotToken string,
) *VoiceMediaClient {
	return &VoiceMediaClient{
		httpClient:       &http.Client{Timeout: voiceDownloadTimeout},
		twilioAccountSID: twilioAccountSID,
		twilioAuthToken:  twilioAuthToken,
		telegramBotToken: telegramBotToken,
		telegramBaseURL:  telegramAPIBaseURL,
	}
}

// Download implements data.VoiceMediaDownloader.
func (c *VoiceMediaClient) Download(
	ctx context.Context,
	channel data.VoiceChannel,
	mediaRef string,
) ([]byte, error) {
	if err := data.ValidateVoiceMediaRef(channel, mediaRef); err != nil {
		return nil, err
	}
	if channel == data.VoiceChannelTelegram {
		return c.downloadTelegram(ctx, mediaRef)
	}
	return c.downloadTwilio(ctx, mediaRef)
}

func (c *VoiceMediaClient) downloadTwilio(ctx context.Context, mediaURL string) ([]byte, error) {
	if c.twilioAccountSID == "" || c.twilioAuthToken == "" {
		return nil, errors.New("twilio credentials are not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create twilio media request: %w", err)
	}
	req.SetBasicAuth(c.twilioAccountSID, c.twilioAuthToken)
	return c.fetch(req)
}

func (c *VoiceMediaClient) downloadTelegram(ctx context.Context, fileID string) ([]byte, error) {
	if c.telegramBotToken == "" {
		return nil, errors.New("telegram bot token is not configured")
	}

	infoURL := fmt.Sprintf(
		"%s/bot%s/getFile?file_id=%s",
		c.telegramBaseURL, c.telegramBotToken, url.QueryEscape(fileID),
	)
	infoReq, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram getFile request: %w", err)
	}
	infoResp, err := c.httpClient.Do(infoReq)
	if err != nil {
		return nil, fmt.Errorf("telegram getFile failed: %w", withoutURL(err))
	}
	defer func() { _ = infoResp.Body.Close() }()
	if infoResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram getFile returned status %d", infoResp.StatusCode)
	}

	var info struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
			FileSize int64  `json:"file_size"`
		} `json:"result"`
	}
	if err := json.NewDecoder(io.LimitReader(infoResp.Body, maxTelegramInfoBytes)).
		Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode telegram getFile response: %w", err)
	}
	if !info.OK || info.Result.FilePath == "" {
		return nil, errors.New("telegram getFile returned no file_path")
	}
	if info.Result.FileSize > data.MaxVoiceBytes {
		return nil, data.ErrVoiceTooLarge
	}

	fileURL := fmt.Sprintf(
		"%s/file/bot%s/%s",
		c.telegramBaseURL,
		c.telegramBotToken,
		info.Result.FilePath,
	)
	fileReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram file request: %w", err)
	}
	return c.fetch(fileReq)
}

// fetch runs req and reads at most data.MaxVoiceBytes of the body.
func (c *VoiceMediaClient) fetch(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("voice download failed: %w", withoutURL(err))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("voice download returned status %d", resp.StatusCode)
	}
	if resp.ContentLength > data.MaxVoiceBytes {
		return nil, data.ErrVoiceTooLarge
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, data.MaxVoiceBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read voice download: %w", withoutURL(err))
	}
	if int64(len(body)) > data.MaxVoiceBytes {
		return nil, data.ErrVoiceTooLarge
	}
	if len(body) == 0 {
		return nil, data.ErrVoiceEmpty
	}
	return body, nil
}

// withoutURL drops the request URL from a transport error. The Telegram URL
// contains the bot token, which must never reach the logs.
func withoutURL(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
