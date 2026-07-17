package handler

import (
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

const errorKey = "error"

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
// @Param		form	body		dto.ExtractionRequest	true	"Extraction Request Form"
// @Success		202		{object}	map[string]string
// @Failure		400		{object}	map[string]string
// @Failure		500		{object}	map[string]string
// @Router		/api/extract [post]
func (h *ApifyHandler) extractMediaHandler(c echo.Context) error {
	req := new(dto.ExtractAPIRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{errorKey: "JSON format not valid"})
	}
	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{errorKey: err.Error()})
	}

	resp, err := h.service.TriggerExtraction(c.Request().Context(), req.URL)
	if err != nil {
		c.Logger().Errorf("TriggerExtraction failed: %v", err)
		return c.JSON(
			http.StatusInternalServerError,
			map[string]string{errorKey: "Failed to process extraction"},
		)
	}

	statusCode := http.StatusOK
	if resp.Status == dto.StatusProcessing {
		statusCode = http.StatusAccepted
	}
	return c.JSON(statusCode, resp)
}

// @Summary		Apify webhook receiver
// @Description	Receives successful run notifications from Apify.
// @Tags		webhooks
// @Accept		json
// @Produce		json
// @Param		payload	body		dto.ApifyWebhookPayload	true	"Apify Webhook Payload"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	map[string]string
// @Router		/api/webhooks/apify [post]
func (h *ApifyHandler) webhookHandler(c echo.Context) error {
	extractionID := c.QueryParam("extractionId")

	payload := new(dto.ApifyWebhookPayload)
	if err := c.Bind(payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{errorKey: "JSON format not valid"})
	}
	if err := c.Validate(payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{errorKey: err.Error()})
	}

	datasetID, err := h.service.ProcessWebhook(c.Request().Context(), payload, extractionID)
	if err != nil {
		c.Logger().Errorf("ProcessWebhook failed (extractionId=%s): %v", extractionID, err)
		return c.JSON(
			http.StatusInternalServerError,
			map[string]string{errorKey: "Internal server error"},
		)
	}

	if datasetID == "" {
		return c.String(http.StatusOK, "Ignored: status is not SUCCEEDED")
	}

	c.Logger().Infof("Apify successful. Dataset ID: %s", datasetID)
	return c.JSON(http.StatusOK, map[string]string{"message": "Webhook received successfully"})
}
