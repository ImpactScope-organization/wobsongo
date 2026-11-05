package repo

import (
	"errors"
	"slices"
	"testing"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
)

func TestFilterBool(t *testing.T) {
	yes, no := true, false

	if got := filterBool(nil); got.Valid {
		t.Errorf("expected nil to mean no filter, got %+v", got)
	}
	if got := filterBool(&yes); !got.Valid || !got.Bool {
		t.Errorf("expected a valid true, got %+v", got)
	}
	if got := filterBool(&no); !got.Valid || got.Bool {
		t.Errorf("expected a valid false, got %+v", got)
	}
}

func TestFilterText(t *testing.T) {
	tests := map[string]struct {
		in        string
		wantValid bool
		want      string
	}{
		"Empty":           {"", false, ""},
		"OnlySpaces":      {"   ", false, ""},
		"Trimmed":         {"  vaccin ", true, "vaccin"},
		"KeepsInnerSpace": {"vaccin  covid", true, "vaccin  covid"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := filterText(tt.in)
			if got.Valid != tt.wantValid || got.String != tt.want {
				t.Errorf(
					"filterText(%q) = %+v, want valid=%v %q",
					tt.in,
					got,
					tt.wantValid,
					tt.want,
				)
			}
		})
	}
}

func TestPageOf(t *testing.T) {
	tests := []struct {
		name          string
		limit, offset int32
		want          int
	}{
		{"FirstPage", 20, 0, 1},
		{"SecondPage", 20, 20, 2},
		{"PartialOffsetRoundsDown", 20, 45, 3},
		{"SmallPages", 5, 5, 2},
		{"ZeroLimit", 0, 0, 1},
		{"NegativeLimit", -1, 10, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pageOf(tt.limit, tt.offset); got != tt.want {
				t.Errorf("pageOf(%d, %d) = %d, want %d", tt.limit, tt.offset, got, tt.want)
			}
		})
	}
}

func TestPaginateAdmin(t *testing.T) {
	active := true
	filter := data.AdminListFilter{IsActive: &active, Search: " vac "}
	query := &dto.PaginationDTO{Page: 3, PerPage: 10}
	times10 := func(n *int) *int {
		v := *n * 10
		return &v
	}

	t.Run("BuildsPageFromRowsAndCount", func(t *testing.T) {
		var gotPage adminPageArgs
		var gotCount adminCountArgs

		got, err := paginateAdmin[int, int](query, filter, "things",
			func(a adminPageArgs) ([]int, error) {
				gotPage = a
				return []int{1, 2}, nil
			},
			func(a adminCountArgs) (int64, error) {
				gotCount = a
				return 22, nil
			},
			times10,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Page != 3 || got.PerPage != 10 || got.TotalItems != 22 ||
			!slices.Equal(got.Items, []int{10, 20}) {
			t.Errorf("unexpected page: %+v", got)
		}
		if gotPage.PageLimit != 10 || gotPage.PageOffset != 20 ||
			!gotPage.IsActive.Valid || !gotPage.IsActive.Bool ||
			!gotPage.Search.Valid || gotPage.Search.String != "vac" {
			t.Errorf("unexpected page args: %+v", gotPage)
		}
		if gotCount.IsActive != gotPage.IsActive || gotCount.Search != gotPage.Search {
			t.Errorf("expected the count to use the same filter, got %+v", gotCount)
		}
	})

	t.Run("ListError_SkipsCount", func(t *testing.T) {
		counted := false

		_, err := paginateAdmin[int, int](query, filter, "things",
			func(adminPageArgs) ([]int, error) { return nil, errors.New("boom") },
			func(adminCountArgs) (int64, error) {
				counted = true
				return 0, nil
			},
			times10,
		)
		if !errors.Is(err, data.ErrInternal) {
			t.Errorf("expected data.ErrInternal, got %v", err)
		}
		if counted {
			t.Error("expected no count after a failed list")
		}
	})

	t.Run("CountError", func(t *testing.T) {
		_, err := paginateAdmin[int, int](query, filter, "things",
			func(adminPageArgs) ([]int, error) { return []int{1}, nil },
			func(adminCountArgs) (int64, error) { return 0, errors.New("boom") },
			times10,
		)
		if !errors.Is(err, data.ErrInternal) {
			t.Errorf("expected data.ErrInternal, got %v", err)
		}
	})
}
