package repo

import (
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// filterBool turns an optional filter into a nullable SQL argument.
func filterBool(v *bool) pgtype.Bool {
	if v == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *v, Valid: true}
}

// filterText turns an optional text filter into a nullable SQL argument. An
// empty text means no filter.
func filterText(s string) pgtype.Text {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// pageOf returns the 1-based page number of an offset.
func pageOf(limit, offset int32) int {
	if limit <= 0 {
		return 1
	}
	return int(offset/limit) + 1
}
