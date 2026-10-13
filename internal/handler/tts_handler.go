package handler

import (
	"errors"
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

const msgTTSFailed = "failed to start speech synthesis"

// TTSHandler handles HTTP requests that turn bot replies into speech.
type TTSHandler struct {
	service *service.SpeechService
}

// NewTTSHandler creates a new handler.
func NewTTSHandler(service *service.SpeechService) *TTSHandler {
	return &TTSHandler{service: service}
}

// @Summary		Start synthesizing a bot reply as a voice message
// @Description	Returns 202 immediately. In the background the text is cleaned
// @Description	(emoji, markdown, citations), translated to English when language is
// @Description	"en", and synthesized; the OGG/Opus audio is then delivered to wa-bot
// @Description	through its /callback/tts-done endpoint under the given jobId.
// @Tags			agent
// @Accept			json
// @Produce		json
// @Param			form	body		dto.TTSRequest	true	"Text to speak"
// @Success		202		{object}	model.APIResponse{data=dto.TTSResponse}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		503		{object}	model.APIResponse{error=string}
// @Router			/api/tts [post]
func (h *TTSHandler) synthesizeHandler(c echo.Context) error {
	req := new(dto.TTSRequest)
	if err := c.Bind(req); err != nil {
		return &model.APIError{
			Code:     http.StatusBadRequest,
			Internal: err,
			Public:   msgInvalidRequestBody,
		}
	}
	if err := c.Validate(req); err != nil {
		return &model.APIError{
			Code:     http.StatusUnprocessableEntity,
			Internal: err,
			Public:   msgValidationFailed,
		}
	}

	if err := h.service.SynthesizeAsync(req.JobID, req.Text, req.Language); err != nil {
		code := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrSpeechNotConfigured):
			code = http.StatusServiceUnavailable
		case errors.Is(err, service.ErrSpeechEmptyText):
			code = http.StatusUnprocessableEntity
		}
		return &model.APIError{Code: code, Internal: err, Public: msgTTSFailed}
	}

	return writeJSON(c, http.StatusAccepted, dto.TTSResponse{
		Status: "processing",
		JobID:  req.JobID,
	})
}
