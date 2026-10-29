package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

const msgInvalidNarrativeID = "invalid narrative id"

// NarrativeHandler handles HTTP requests related to narratives.
type NarrativeHandler struct {
	service *service.NarrativeService
}

// NewNarrativeHandler creates a new handler.
func NewNarrativeHandler(service *service.NarrativeService) *NarrativeHandler {
	return &NarrativeHandler{service: service}
}

// @Summary		List narratives
// @Description	Retrieves a paginated list of narratives, newest first.
// @Tags			narratives
// @Produce		json
// @Param			page		query	int	false	"Page number"		minimum(1)
// @Param			per_page	query	int	false	"Items per page"	minimum(1)	maximum(100)
// @Success		200		{object}	model.APIResponse{data=dto.PaginationResults[model.Narrative]}
// @Failure		400		{object}	model.APIResponse{error=string}
// @Failure		401		{object}	model.APIResponse{error=string}
// @Failure		422		{object}	model.APIResponse{error=string}
// @Failure		500		{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/narratives [get]
func (h *NarrativeHandler) listNarrativesHandler(c echo.Context) error {
	pagination := new(dto.PaginationDTO)
	if err := c.Bind(pagination); err != nil {
		return &model.APIError{
			Code:     http.StatusBadRequest,
			Internal: err,
			Public:   msgInvalidQueryParams,
		}
	}
	if err := c.Validate(pagination); err != nil {
		return &model.APIError{
			Code:     http.StatusUnprocessableEntity,
			Internal: err,
			Public:   msgValidationFailed,
		}
	}

	results, err := h.service.List(c.Request().Context(), pagination)
	if err != nil {
		status, message := mapDataErrorToHTTP(err)
		return &model.APIError{Code: status, Internal: err, Public: message}
	}

	return writeJSON(c, http.StatusOK, results)
}

// @Summary		Get a narrative by ID
// @Description	Retrieves a single narrative by its ID.
// @Tags			narratives
// @Produce		json
// @Param			id	path	string	true	"Narrative ID"	format(uuid)
// @Success		200	{object}	model.APIResponse{data=model.Narrative}
// @Failure		400	{object}	model.APIResponse{error=string}
// @Failure		401	{object}	model.APIResponse{error=string}
// @Failure		404	{object}	model.APIResponse{error=string}
// @Failure		500	{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/narratives/{id} [get]
func (h *NarrativeHandler) getNarrativeHandler(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return &model.APIError{
			Code:     http.StatusBadRequest,
			Internal: err,
			Public:   msgInvalidNarrativeID,
		}
	}

	narrative, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		status, message := mapDataErrorToHTTP(err)
		return &model.APIError{Code: status, Internal: err, Public: message}
	}

	return writeJSON(c, http.StatusOK, narrative)
}
