package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

const (
	msgAgentInboundFailed      = "failed to process inbound message"
	msgAgentInboundVoiceFailed = "failed to process inbound voice message"

	// maxVoiceUploadBytes caps an uploaded voice message (16 MB).
	maxVoiceUploadBytes = 16 * 1024 * 1024
)

// AgentHandler handles HTTP requests for the agentic bot workflow
type AgentHandler struct {
	service *service.AgentService
}

// NewAgentHandler creates a new handler.
func NewAgentHandler(service *service.AgentService) *AgentHandler {
	return &AgentHandler{
		service: service,
	}
}

// @Summary		Handle an inbound WhatsApp message
// @Description	Single entry point for every inbound WhatsApp message. Applies a
// @Description	deterministic fast path for bare TikTok URLs (straight to the existing
// @Description	cache-check/Apify pipeline) and routes everything else through the
// @Description	conversational agent (tool-calling over claim-checking and video
// @Description	processing, with per-jid conversation history).
// @Tags			agent
// @Accept			json
// @Produce		json
// @Param			form	body		dto.AgentInboundRequest	true	"Inbound Message"
// @Success		202		{object}	model.APIResponse{data=dto.AgentInboundResponse}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Router			/api/agent/inbound [post]
func (h *AgentHandler) inboundMessageHandler(c echo.Context) error {
	req := new(dto.AgentInboundRequest)
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

	resp, err := h.service.HandleInboundMessage(
		c.Request().Context(),
		req.Jid,
		req.Text,
	)
	if err != nil {
		return &model.APIError{
			Code:     http.StatusInternalServerError,
			Internal: err,
			Public:   msgAgentInboundFailed,
		}
	}

	return writeJSON(c, http.StatusAccepted, resp)
}

// @Summary		Handle an inbound voice message
// @Description	Receives a voice message, transcribes it in the
// @Description	background and routes the transcript through the conversational
// @Description	agent. The answer is delivered through the same callback as /agent/inbound.
// @Tags			agent
// @Accept			mpfd
// @Produce		json
// @Param			jid		formData	string	true	"Chat identifier"
// @Param			format	formData	string	true	"Audio format hint (e.g. ogg)"
// @Param			audio	formData	file	true	"Voice message audio"
// @Success		202		{object}	model.APIResponse{data=dto.AgentInboundResponse}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		413		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Router			/api/agent/inbound-voice [post]
func (h *AgentHandler) inboundVoiceHandler(c echo.Context) error {
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, maxVoiceUploadBytes)

	req := new(dto.AgentInboundVoiceRequest)
	if err := c.Bind(req); err != nil {
		return voiceUploadError(err)
	}
	if err := c.Validate(req); err != nil {
		return &model.APIError{
			Code:     http.StatusUnprocessableEntity,
			Internal: err,
			Public:   msgValidationFailed,
		}
	}

	fileHeader, err := c.FormFile("audio")
	if err != nil {
		return voiceUploadError(err)
	}
	file, err := fileHeader.Open()
	if err != nil {
		return voiceUploadError(err)
	}
	defer func() { _ = file.Close() }()

	audio, err := io.ReadAll(file)
	if err != nil {
		return voiceUploadError(err)
	}
	if len(audio) == 0 {
		return &model.APIError{
			Code:     http.StatusUnprocessableEntity,
			Internal: errors.New("empty audio"),
			Public:   msgValidationFailed,
		}
	}

	resp, err := h.service.HandleInboundVoice(c.Request().Context(), req.JID, req.Format, audio)
	if err != nil {
		return &model.APIError{
			Code:     http.StatusInternalServerError,
			Internal: err,
			Public:   msgAgentInboundVoiceFailed,
		}
	}
	return writeJSON(c, http.StatusAccepted, resp)
}

// voiceUploadError maps an upload read/parse error to 413 (too large) or 400.
func voiceUploadError(err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return &model.APIError{
			Code:     http.StatusRequestEntityTooLarge,
			Internal: err,
			Public:   "voice message is too large",
		}
	}
	return &model.APIError{
		Code:     http.StatusBadRequest,
		Internal: err,
		Public:   msgInvalidRequestBody,
	}
}
