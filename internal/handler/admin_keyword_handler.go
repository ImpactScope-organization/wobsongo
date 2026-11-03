package handler

import (
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

// AdminKeywordHandler handles the admin management of discovery keywords.
type AdminKeywordHandler struct {
	service *service.AdminKeywordService
}

// NewAdminKeywordHandler creates a new handler.
func NewAdminKeywordHandler(service *service.AdminKeywordService) *AdminKeywordHandler {
	return &AdminKeywordHandler{service: service}
}

// @Summary		List discovery keywords
// @Description	Retrieves a paginated list of discovery keywords.
// @Tags			admin
// @Produce		json
// @Param			page		query		int		false	"Page number"		minimum(1)
// @Param			per_page	query		int		false	"Items per page"	minimum(1)	maximum(100)
// @Param			is_active	query		bool	false	"Filter by active state"
// @Param			search		query		string	false	"Text the keyword contains"
// @Success		200			{object}	model.APIResponse{data=dto.PaginationResults[model.DiscoveryKeyword]}
// @Failure		400			{object}	model.APIResponse{error=string}
// @Failure		401			{object}	model.APIResponse{error=string}
// @Failure		403			{object}	model.APIResponse{error=string}
// @Failure		422			{object}	model.APIResponse{error=string}
// @Failure		500			{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/keywords [get]
func (h *AdminKeywordHandler) listKeywordsHandler(c echo.Context) error {
	query := new(dto.KeywordListDTO)
	if err := bindQueryAndValidate(c, query); err != nil {
		return err
	}

	results, err := h.service.List(c.Request().Context(), query)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusOK, results)
}

// @Summary		Get a discovery keyword
// @Tags			admin
// @Produce		json
// @Param			id	path		string	true	"Keyword ID"	format(uuid)
// @Success		200	{object}	model.APIResponse{data=model.DiscoveryKeyword}
// @Failure		400	{object}	model.APIResponse{error=string}
// @Failure		401	{object}	model.APIResponse{error=string}
// @Failure		403	{object}	model.APIResponse{error=string}
// @Failure		404	{object}	model.APIResponse{error=string}
// @Failure		500	{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/keywords/{id} [get]
func (h *AdminKeywordHandler) getKeywordHandler(c echo.Context) error {
	id, err := parseIDParam(c, msgInvalidKeywordID)
	if err != nil {
		return err
	}

	keyword, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusOK, keyword)
}

// @Summary		Create a discovery keyword
// @Description	Adds a keyword used to discover new videos. The same keyword in the same language cannot be added twice.
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			form	body		dto.CreateKeywordRequest	true	"Keyword"
// @Success		201		{object}	model.APIResponse{data=model.DiscoveryKeyword}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		403		{object}	model.APIResponse{error=string}
// @Failure		409		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/keywords [post]
func (h *AdminKeywordHandler) createKeywordHandler(c echo.Context) error {
	req := new(dto.CreateKeywordRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}

	keyword, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusCreated, keyword)
}

// @Summary		Update a discovery keyword
// @Description	Replaces the keyword, language and active state.
// @Tags			admin
// @Accept			json
// @Produce		json
// @Param			id		path		string						true	"Keyword ID"	format(uuid)
// @Param			form	body		dto.UpdateKeywordRequest	true	"Keyword"
// @Success		200		{object}	model.APIResponse{data=model.DiscoveryKeyword}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		403		{object}	model.APIResponse{error=string}
// @Failure		404		{object}	model.APIResponse{error=string}
// @Failure		409		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/keywords/{id} [put]
func (h *AdminKeywordHandler) updateKeywordHandler(c echo.Context) error {
	id, err := parseIDParam(c, msgInvalidKeywordID)
	if err != nil {
		return err
	}

	req := new(dto.UpdateKeywordRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}

	keyword, err := h.service.Update(c.Request().Context(), id, req)
	if err != nil {
		return dataAPIError(err)
	}

	return writeJSON(c, http.StatusOK, keyword)
}

// @Summary		Delete a discovery keyword
// @Tags			admin
// @Param			id	path	string	true	"Keyword ID"	format(uuid)
// @Success		204
// @Failure		400	{object}	model.APIResponse{error=string}
// @Failure		401	{object}	model.APIResponse{error=string}
// @Failure		403	{object}	model.APIResponse{error=string}
// @Failure		404	{object}	model.APIResponse{error=string}
// @Failure		500	{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/admin/keywords/{id} [delete]
func (h *AdminKeywordHandler) deleteKeywordHandler(c echo.Context) error {
	id, err := parseIDParam(c, msgInvalidKeywordID)
	if err != nil {
		return err
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		return dataAPIError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
