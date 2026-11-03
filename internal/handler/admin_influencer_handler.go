package handler

import (
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

// AdminInfluencerHandler handles the admin management of monitored influencers.
type AdminInfluencerHandler struct {
	service *service.AdminInfluencerService
}

// NewAdminInfluencerHandler creates a new handler.
func NewAdminInfluencerHandler(service *service.AdminInfluencerService) *AdminInfluencerHandler {
	return &AdminInfluencerHandler{service: service}
}

// @Summary		List influencers
// @Description	Retrieves a paginated list of influencers, newest first.
// @Tags			admin
// @Produce		json
// @Param			page		query		int		false	"Page number"		minimum(1)
// @Param			per_page	query		int		false	"Items per page"	minimum(1)	maximum(100)
// @Param			is_active	query		bool	false	"Filter by monitored state"
// @Param			source		query		string	false	"Filter by source"	Enums(seed, discovered)
// @Param			search		query		string	false	"Text the username or nickname contains"
// @Success		200			{object}	model.APIResponse{data=dto.PaginationResults[model.Influencer]}
// @Failure		400			{object}	model.APIResponse{error=string}
// @Failure		401			{object}	model.APIResponse{error=string}
// @Failure		403			{object}	model.APIResponse{error=string}
// @Failure		422			{object}	model.APIResponse{error=string}
// @Failure		500			{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/influencers [get]
func (h *AdminInfluencerHandler) listInfluencersHandler(c echo.Context) error {
	query := new(dto.InfluencerListDTO)
	if err := bindQueryAndValidate(c, query); err != nil {
		return err
	}

	results, err := h.service.List(c.Request().Context(), query)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusOK, results)
}

// @Summary		Get an influencer
// @Tags			admin
// @Produce		json
// @Param			id	path		string	true	"Influencer ID"	format(uuid)
// @Success		200	{object}	model.APIResponse{data=model.Influencer}
// @Failure		400	{object}	model.APIResponse{error=string}
// @Failure		401	{object}	model.APIResponse{error=string}
// @Failure		403	{object}	model.APIResponse{error=string}
// @Failure		404	{object}	model.APIResponse{error=string}
// @Failure		500	{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/influencers/{id} [get]
func (h *AdminInfluencerHandler) getInfluencerHandler(c echo.Context) error {
	id, err := parseIDParam(c, msgInvalidInfluencerID)
	if err != nil {
		return err
	}

	influencer, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusOK, influencer)
}

// @Summary		Add an influencer
// @Description	Adds a TikTok influencer to monitor. Profile details are filled by the next monitor run, which starts right away. A username that already exists (ignoring case) gives 409.
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			form	body		dto.CreateInfluencerRequest	true	"Influencer"
// @Success		201		{object}	model.APIResponse{data=model.Influencer}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		403		{object}	model.APIResponse{error=string}
// @Failure		409		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/influencers [post]
func (h *AdminInfluencerHandler) createInfluencerHandler(c echo.Context) error {
	req := new(dto.CreateInfluencerRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}

	influencer, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusCreated, influencer)
}

// @Summary		Turn influencer monitoring on or off
// @Description	Turning monitoring on again resets the empty-check counter and schedules a check right away.
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			id		path		string							true	"Influencer ID"	format(uuid)
// @Param			form	body		dto.SetInfluencerActiveRequest	true	"Active state"
// @Success		200		{object}	model.APIResponse{data=model.Influencer}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		403		{object}	model.APIResponse{error=string}
// @Failure		404		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/influencers/{id} [patch]
func (h *AdminInfluencerHandler) setInfluencerActiveHandler(c echo.Context) error {
	id, err := parseIDParam(c, msgInvalidInfluencerID)
	if err != nil {
		return err
	}

	req := new(dto.SetInfluencerActiveRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}

	influencer, err := h.service.SetActive(c.Request().Context(), id, *req.IsActive)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusOK, influencer)
}
