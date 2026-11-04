package handler

import (
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// dataAPIError turns a service error into the API error of its HTTP status.
func dataAPIError(err error) error {
	status, message := mapDataErrorToHTTP(err)
	return &model.APIError{Code: status, Internal: err, Public: message}
}
