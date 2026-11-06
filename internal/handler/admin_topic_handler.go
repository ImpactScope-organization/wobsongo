package handler

import (
	"errors"
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

// AdminTopicHandler handles the admin management of claim topics.
type AdminTopicHandler struct {
	service *service.AdminTopicService
}

// NewAdminTopicHandler creates a new handler.
func NewAdminTopicHandler(service *service.AdminTopicService) *AdminTopicHandler {
	return &AdminTopicHandler{service: service}
}

// @Summary		List topics
// @Description	Retrieves a paginated list of topics, active or not.
// @Tags			admin
// @Produce		json
// @Param			page		query		int		false	"Page number"		minimum(1)
// @Param			per_page	query		int		false	"Items per page"	minimum(1)	maximum(100)
// @Param			is_active	query		bool	false	"Filter by active state"
// @Param			search		query		string	false	"Text the name contains"
// @Success		200			{object}	model.APIResponse{data=dto.PaginationResults[model.Topic]}
// @Failure		400			{object}	model.APIResponse{error=string}
// @Failure		401			{object}	model.APIResponse{error=string}
// @Failure		403			{object}	model.APIResponse{error=string}
// @Failure		422			{object}	model.APIResponse{error=string}
// @Failure		500			{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/topics [get]
func (h *AdminTopicHandler) listTopicsHandler(c echo.Context) error {
	query := new(dto.TopicListDTO)
	if err := bindAndValidate(c, query); err != nil {
		return err
	}

	results, err := h.service.List(c.Request().Context(), query)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusOK, results)
}

// @Summary		Get a topic
// @Tags			admin
// @Produce		json
// @Param			id	path		string	true	"Topic ID"	format(uuid)
// @Success		200	{object}	model.APIResponse{data=model.Topic}
// @Failure		400	{object}	model.APIResponse{error=string}
// @Failure		401	{object}	model.APIResponse{error=string}
// @Failure		403	{object}	model.APIResponse{error=string}
// @Failure		404	{object}	model.APIResponse{error=string}
// @Failure		500	{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/topics/{id} [get]
func (h *AdminTopicHandler) getTopicHandler(c echo.Context) error {
	id, err := parseIDParam(c, msgInvalidTopicID)
	if err != nil {
		return err
	}

	topic, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusOK, topic)
}

// @Summary		Create a topic
// @Description	Adds a topic the claim analyzer may assign. The name must be unique.
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			form	body		dto.CreateTopicRequest	true	"Topic"
// @Success		201		{object}	model.APIResponse{data=model.Topic}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		403		{object}	model.APIResponse{error=string}
// @Failure		409		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/topics [post]
func (h *AdminTopicHandler) createTopicHandler(c echo.Context) error {
	req := new(dto.CreateTopicRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}

	topic, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusCreated, topic)
}

// @Summary		Update a topic
// @Description	Replaces name, description and active state. Topics are never deleted; set is_active to false instead. The last active topic cannot be deactivated (409).
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			id		path		string					true	"Topic ID"	format(uuid)
// @Param			form	body		dto.UpdateTopicRequest	true	"Topic"
// @Success		200		{object}	model.APIResponse{data=model.Topic}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		403		{object}	model.APIResponse{error=string}
// @Failure		404		{object}	model.APIResponse{error=string}
// @Failure		409		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/topics/{id} [put]
func (h *AdminTopicHandler) updateTopicHandler(c echo.Context) error {
	id, err := parseIDParam(c, msgInvalidTopicID)
	if err != nil {
		return err
	}

	req := new(dto.UpdateTopicRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}

	topic, err := h.service.Update(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, service.ErrLastActiveTopic) {
			return &model.APIError{
				Code:     http.StatusConflict,
				Internal: err,
				Public:   msgLastActiveTopic,
			}
		}
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusOK, topic)
}
