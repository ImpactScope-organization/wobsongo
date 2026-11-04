package repo

import (
	"fmt"
	"strings"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
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

// adminPageArgs holds the arguments for an admin page query.
type adminPageArgs struct {
	IsActive   pgtype.Bool
	Search     pgtype.Text
	PageOffset int32
	PageLimit  int32
}

// adminCountArgs are the arguments of an admin count query. 
type adminCountArgs struct {
	IsActive pgtype.Bool
	Search   pgtype.Text
}

// paginateAdmin runs the page and count queries of an admin list endpoint and
// assembles the page. entity names the listed rows in error messages.
func paginateAdmin[Row, Model any](
	q data.SupportsPagination,
	filter data.AdminListFilter,
	entity string,
	list func(adminPageArgs) ([]Row, error),
	count func(adminCountArgs) (int64, error),
	toModel func(*Row) *Model,
) (*dto.PaginationResults[Model], error) {
	limit, offset := q.Limit(), q.Offset()
	isActive, search := filterBool(filter.IsActive), filterText(filter.Search)

	rows, err := list(adminPageArgs{
		IsActive:   isActive,
		Search:     search,
		PageOffset: offset,
		PageLimit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", entity, mapPostgresError(err))
	}

	total, err := count(adminCountArgs{IsActive: isActive, Search: search})
	if err != nil {
		return nil, fmt.Errorf("failed to count %s: %w", entity, mapPostgresError(err))
	}

	items := make([]Model, 0, len(rows))
	for i := range rows {
		items = append(items, *toModel(&rows[i]))
	}

	return &dto.PaginationResults[Model]{
		Page:       pageOf(limit, offset),
		PerPage:    int(limit),
		TotalItems: int(total),
		Items:      items,
	}, nil
}
