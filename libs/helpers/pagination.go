package helpers

import (
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

type PaginatedResult struct {
	Items      any   `json:"items"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

var columnRe = regexp.MustCompile(`^[a-z_]+(\.[a-z_]+)?$`)

// BuildPaginatedQuery menerapkan filter, sort dan pagination.
// Key filter berformat "<column>.<op>", mis. "payments.status.eq", "payments.deleted_at.isnull".
// Op: eq, ne, gt, gte, lt, lte, like, ilike, in, isnull, notnull. Value nil diabaikan (kecuali isnull/notnull).
// Sort: "created_at" atau "-created_at" (desc).
func BuildPaginatedQuery(q *gorm.DB, page, limit int, sort, defaultSort string, filters map[string]any, out any) (*PaginatedResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	for k, v := range filters {
		i := strings.LastIndex(k, ".")
		if i < 0 || !columnRe.MatchString(k[:i]) {
			return nil, fmt.Errorf("invalid filter %q", k)
		}
		col, op := k[:i], k[i+1:]
		switch op {
		case "isnull":
			q = q.Where(col + " IS NULL")
			continue
		case "notnull":
			q = q.Where(col + " IS NOT NULL")
			continue
		}
		if isEmpty(v) {
			continue
		}
		switch op {
		case "eq":
			q = q.Where(col+" = ?", v)
		case "ne":
			q = q.Where(col+" <> ?", v)
		case "gt":
			q = q.Where(col+" > ?", v)
		case "gte":
			q = q.Where(col+" >= ?", v)
		case "lt":
			q = q.Where(col+" < ?", v)
		case "lte":
			q = q.Where(col+" <= ?", v)
		case "like":
			q = q.Where(col+" LIKE ?", "%"+fmt.Sprint(v)+"%")
		case "ilike":
			q = q.Where(col+" ILIKE ?", "%"+fmt.Sprint(v)+"%")
		case "in":
			q = q.Where(col+" IN ?", v)
		default:
			return nil, fmt.Errorf("invalid filter op %q", op)
		}
	}

	var total int64
	if e := q.Count(&total).Error; e != nil {
		return nil, e
	}

	order := defaultSort
	if sort != "" {
		dir, col := "ASC", sort
		if strings.HasPrefix(sort, "-") {
			dir, col = "DESC", sort[1:]
		}
		if !columnRe.MatchString(col) {
			return nil, fmt.Errorf("invalid sort %q", sort)
		}
		order = col + " " + dir
	}
	if e := q.Order(order).Offset((page - 1) * limit).Limit(limit).Find(out).Error; e != nil {
		return nil, e
	}
	return &PaginatedResult{
		Items:      out,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	}, nil
}

func isEmpty(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	case *string:
		return t == nil || *t == ""
	case *bool:
		return t == nil
	}
	return false
}
