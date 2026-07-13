package handler

import (
	"log"
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
	req := new(dto.ExtractionRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{errorKey: "JSON format not valid"})
	}

	if err := h.service.TriggerExtraction(c.Request().Context(), req); err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			map[string]string{errorKey: "Failed to enqueue media extraction job"},
		)
	}

	return c.JSON(
		http.StatusAccepted,
		map[string]string{"message": "Successfully enqueued media extraction job."},
	)
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
	payload := new(dto.ApifyWebhookPayload)
	if err := c.Bind(payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{errorKey: "JSON format not valid"})
	}

	datasetID, err := h.service.ProcessWebhook(c.Request().Context(), payload)
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			map[string]string{errorKey: "Internal server error"},
		)
	}

	if datasetID == "" {
		return c.String(http.StatusOK, "Ignored: status is not SUCCEEDED")
	}

	log.Printf("Apify successful. Dataset ID: %s\n", datasetID)
	return c.JSON(http.StatusOK, map[string]string{"message": "Webhook received successfully"})
}
