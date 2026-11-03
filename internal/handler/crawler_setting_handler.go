package handler

import (
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

// CrawlerSettingHandler handles the admin crawler configuration.
type CrawlerSettingHandler struct {
	service *service.CrawlSettingService
}

// NewCrawlerSettingHandler creates a new handler.
func NewCrawlerSettingHandler(service *service.CrawlSettingService) *CrawlerSettingHandler {
	return &CrawlerSettingHandler{service: service}
}

// @Summary		Get crawler settings
// @Description	The configuration of video discovery and creator monitoring.
// @Tags			crawler
// @Produce		json
// @Success		200	{object}	model.APIResponse{data=model.CrawlerSetting}
// @Failure		401	{object}	model.APIResponse{error=string}
// @Failure		403	{object}	model.APIResponse{error=string}
// @Failure		500	{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/crawler/settings [get]
func (h *CrawlerSettingHandler) getSettingHandler(c echo.Context) error {
	setting, err := h.service.Get(c.Request().Context())
	if err != nil {
		status, message := mapDataErrorToHTTP(err)
		return &model.APIError{Code: status, Internal: err, Public: message}
	}

	return writeJSON(c, http.StatusOK, setting)
}

// @Summary		Update crawler settings
// @Description	Replaces the whole configuration. A new discovery interval replaces the periodic discovery job right away. max_cost_per_run is stored but not enforced yet.
// @Tags			crawler
// @Accept			json
// @Produce		json
// @Param			form	body		dto.UpdateCrawlerSettingRequest	true	"Crawler settings"
// @Success		200		{object}	model.APIResponse{data=model.CrawlerSetting}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		403		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/crawler/settings [put]
func (h *CrawlerSettingHandler) updateSettingHandler(c echo.Context) error {
	req := new(dto.UpdateCrawlerSettingRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}

	setting, err := h.service.Update(
		c.Request().Context(), ClaimsFromContext(c).UserID, req,
	)
	if err != nil {
		status, message := mapDataErrorToHTTP(err)
		return &model.APIError{Code: status, Internal: err, Public: message}
	}

	return writeJSON(c, http.StatusOK, setting)
}
