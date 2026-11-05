package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/labstack/echo/v4"
)

const msgInvalidKeywordID = "invalid keyword id"

// parseIDParam reads the :id path parameter as a UUID.
func parseIDParam(c echo.Context, message string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, &model.APIError{
			Code:     http.StatusBadRequest,
			Internal: err,
			Public:   message,
		}
	}
	return id, nil
}

// dataAPIError turns a service error into the API error of its HTTP status.
func dataAPIError(err error) error {
	status, message := mapDataErrorToHTTP(err)
	return &model.APIError{Code: status, Internal: err, Public: message}
}
