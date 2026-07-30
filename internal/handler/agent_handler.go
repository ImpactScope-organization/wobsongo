package handler

import (
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

const msgAgentInboundFailed = "failed to process inbound message"

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
