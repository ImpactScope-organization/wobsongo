package handler

import (
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

const (
	msgInvalidRequestBody = "invalid request body"
	msgExtractionFailed   = "failed to process extraction"
	msgWebhookFailed      = "failed to process webhook"
)

// apifyHandler handles HTTP requests related to Apify extraction and webhooks.
type ApifyHandler struct {
	service *service.ApifyService
}

// NewApifyHandler creates a new handler.
func NewApifyHandler(service *service.ApifyService) *ApifyHandler {
	return &ApifyHandler{
		service: service,
	}
}

// @Summary		Trigger media extraction
// @Description	Enqueues a job to extract media from a target URL using Apify.
// @Tags		media
// @Accept		json
// @Produce		json
// @Param		form	body		dto.ExtractAPIRequest	true	"Extraction Request Form"
// @Success		202		{object}	model.APIResponse{data=dto.ExtractResponse}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Router		/api/extract [post]
func (h *ApifyHandler) extractMediaHandler(c echo.Context) error {
	req := new(dto.ExtractAPIRequest)
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

	resp, err := h.service.TriggerExtraction(c.Request().Context(), req.URL, req.Question)
	if err != nil {
		return &model.APIError{
			Code:     http.StatusInternalServerError,
			Internal: err,
			Public:   msgExtractionFailed,
		}
	}

	statusCode := http.StatusOK
	if resp.Status == dto.StatusProcessing {
		statusCode = http.StatusAccepted
	}
	return writeJSON(c, statusCode, resp)
}

// @Summary		Apify webhook receiver
// @Description	Receives successful run notifications from Apify.
// @Tags		webhooks
// @Accept		json
// @Produce		json
// @Param		payload	body		dto.ApifyWebhookPayload	true	"Apify Webhook Payload"
// @Success		200		{object}	model.APIResponse{data=string}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Router		/api/webhooks/apify [post]
func (h *ApifyHandler) webhookHandler(c echo.Context) error {
	extractionID := c.QueryParam("extractionId")

	payload := new(dto.ApifyWebhookPayload)
	if err := c.Bind(payload); err != nil {
		return &model.APIError{
			Code:     http.StatusBadRequest,
			Internal: err,
			Public:   msgInvalidRequestBody,
		}
	}
	if err := c.Validate(payload); err != nil {
		return &model.APIError{
			Code:     http.StatusUnprocessableEntity,
			Internal: err,
			Public:   msgValidationFailed,
		}
	}

	datasetID, err := h.service.ProcessWebhook(c.Request().Context(), payload, extractionID)
	if err != nil {
		return &model.APIError{
			Code:     http.StatusInternalServerError,
			Internal: err,
			Public:   msgWebhookFailed,
		}
	}

	if datasetID == "" {
		return c.String(http.StatusOK, "Ignored: status is not SUCCEEDED")
	}

	return writeJSON(c, http.StatusOK, "webhook received successfully")
}
