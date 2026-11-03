package handler

import (
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

// AdminSettingHandler handles the admin settings of video discovery and
// creator monitoring.
type AdminSettingHandler struct {
	service *service.AdminSettingService
}

// NewAdminSettingHandler creates a new handler.
func NewAdminSettingHandler(service *service.AdminSettingService) *AdminSettingHandler {
	return &AdminSettingHandler{service: service}
}

// @Summary		Get admin settings
// @Description	The settings of video discovery and creator monitoring.
// @Tags			admin
// @Produce		json
// @Success		200	{object}	model.APIResponse{data=model.AdminSetting}
// @Failure		401	{object}	model.APIResponse{error=string}
// @Failure		403	{object}	model.APIResponse{error=string}
// @Failure		500	{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/settings [get]
func (h *AdminSettingHandler) getSettingHandler(c echo.Context) error {
	setting, err := h.service.Get(c.Request().Context())
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusOK, setting)
}

// @Summary		Update admin settings
// @Description	Replaces the whole configuration. A new discovery interval replaces the periodic discovery job right away. max_cost_per_run is stored but not enforced yet.
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			form	body		dto.UpdateAdminSettingRequest	true	"Admin settings"
// @Success		200		{object}	model.APIResponse{data=model.AdminSetting}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		403		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/settings [put]
func (h *AdminSettingHandler) updateSettingHandler(c echo.Context) error {
	req := new(dto.UpdateAdminSettingRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}

	setting, err := h.service.Update(c.Request().Context(), ClaimsFromContext(c).UserID, req)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusOK, setting)
}
