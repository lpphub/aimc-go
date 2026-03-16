package pagination

import "gorm.io/gorm"

/*
Offset / 传统分页
*/

type Offset struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

func (p *Offset) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
}

type OffsetPageData[T any] struct {
	Total int64 `json:"total"`
	List  []T   `json:"list"`
}

// OffsetScope GORM Scope
func OffsetScope(p Offset) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		p.Normalize()
		offset := (p.Page - 1) * p.PageSize
		return db.Offset(offset).Limit(p.PageSize)
	}
}

// QueryOffset 执行分页
func QueryOffset[T any](db *gorm.DB, p Offset) (*OffsetPageData[T], error) {
	var (
		total int64
		list  []T
	)
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	if total > 0 {
		if err := db.Scopes(OffsetScope(p)).Find(&list).Error; err != nil {
			return nil, err
		}
	}
	return WithOffsetPageData(total, list), nil
}

func WithOffsetPageData[T any](total int64, list []T) *OffsetPageData[T] {
	return &OffsetPageData[T]{
		Total: total,
		List:  list,
	}
}