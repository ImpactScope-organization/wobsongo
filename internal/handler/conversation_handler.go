package handler

import (
	"net/http"

	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/labstack/echo/v4"
)

// ConversationHandler handles HTTP requests related to the bot's conversations.
type ConversationHandler struct {
	service *service.ConversationService
}

// NewConversationHandler creates a new handler.
func NewConversationHandler(service *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{service: service}
}

// @Summary		User activity
// @Description	Counts the distinct users and the messages they sent over the last 7, 15 or 30 days. Left out, all time is counted.
// @Tags			conversations
// @Produce		json
// @Param			days	query	int	false	"Count only the last N days; left out, all time"	Enums(7, 15, 30)
// @Success		200	{object}	model.APIResponse{data=model.UserActivity}
// @Failure		400	{object}	model.APIResponse{error=string}
// @Failure		401	{object}	model.APIResponse{error=string}
// @Failure		422	{object}	model.APIResponse{error=string}
// @Failure		500	{object}	model.APIResponse{error=string}
// @Security	BearerAuth
// @Router			/api/v1/conversations/stats [get]
func (h *ConversationHandler) statsHandler(c echo.Context) error {
	query := new(dto.UserActivityDTO)
	if err := c.Bind(query); err != nil {
		return &model.APIError{
			Code:     http.StatusBadRequest,
			Internal: err,
			Public:   msgInvalidQueryParams,
		}
	}
	if err := c.Validate(query); err != nil {
		return &model.APIError{
			Code:     http.StatusUnprocessableEntity,
			Internal: err,
			Public:   msgValidationFailed,
		}
	}

	activity, err := h.service.UserActivity(c.Request().Context(), query)
	if err != nil {
		status, message := mapDataErrorToHTTP(err)
		return &model.APIError{Code: status, Internal: err, Public: message}
	}

	return writeJSON(c, http.StatusOK, activity)
}
