package pagination

import (
	"encoding/base64"
	"encoding/json"

	"gorm.io/gorm"
)

/*
Cursor / 瀑布流分页
------------------------------------------------
特性：
- 首次 cursor 为空自动返回第一页
- 强类型 cursor（JSON + base64）
- 无 offset，性能稳定
*/

const (
	defaultLimit = 20
	maxLimit     = 50
)

type Cursor struct {
	Limit  int    `json:"limit"  form:"limit"`
	Cursor string `json:"cursor" form:"cursor"`
}

func (p *Cursor) Normalize() {
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
}

type CursorPageData[T any] struct {
	List       []T    `json:"list"`
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor,omitempty"`
}

/*
------------------------------------------------
CursorStrategy
------------------------------------------------

CursorStrategy 描述了一套游标分页策略，由三个阶段组成：

1. Order：定义结果集的排序规则
2. Filter：根据 cursor 追加分页过滤条件（WHERE）
3. Next ：根据当前页最后一条记录生成下一页 cursor

约束：
- Order、Filter、Next 必须基于完全一致的字段与排序方向
- Order 中使用的字段必须全部包含在 cursor 中
- Filter 的比较逻辑必须与 Order 的排序方向严格一致

CursorStrategy 本身无状态，仅描述分页行为。
*/

type CursorStrategy[T any] struct {
	// Order 定义分页所需的排序规则（必须和 cursor 一致）
	Order func(db *gorm.DB) *gorm.DB

	// Filter 根据 cursor 追加分页查询条件（WHERE）
	// cursor 为空或非法时应返回原 DB
	Filter func(db *gorm.DB, cursor string) *gorm.DB

	// Next 根据一条记录生成下一页的 cursor
	Next func(item T) (string, error)
}

/*
------------------------------------------------
统一执行入口
------------------------------------------------
*/

func QueryCursor[T any](
	db *gorm.DB,
	q Cursor,
	strategy CursorStrategy[T],
) (*CursorPageData[T], error) {

	q.Normalize()

	// 1. apply order
	if strategy.Order != nil {
		db = strategy.Order(db)
	}

	// 2. apply cursor filter
	if q.Cursor != "" && strategy.Filter != nil {
		db = strategy.Filter(db, q.Cursor)
	}

	var list []T
	if err := db.Limit(q.Limit + 1).Find(&list).Error; err != nil {
		return nil, err
	}

	hasMore := false
	if len(list) > q.Limit {
		hasMore = true
		list = list[:q.Limit]
	}

	var nextCursor string
	if hasMore && len(list) > 0 && strategy.Next != nil {
		if c, err := strategy.Next(list[len(list)-1]); err == nil {
			nextCursor = c
		}
	}

	return &CursorPageData[T]{
		List:       list,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

/*
================================================
Cursor 编解码（强类型）
================================================
*/

func encodeCursor(v any) string {
	b, _ := json.Marshal(v)
	return base64.StdEncoding.EncodeToString(b)
}

func decodeCursor[T any](cursor string) (T, bool) {
	var zero T
	if cursor == "" {
		return zero, false
	}

	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return zero, false
	}

	if err := json.Unmarshal(b, &zero); err != nil {
		return zero, false
	}
	return zero, true
}

/*
================================================
单字段 Cursor（简单场景）
================================================
*/

func OrderBy[T any, V comparable](
	field string,
	desc bool,
	getValue func(T) V,
) CursorStrategy[T] {

	order := "ASC"
	op := ">"
	if desc {
		order = "DESC"
		op = "<"
	}

	return CursorStrategy[T]{
		Order: func(db *gorm.DB) *gorm.DB {
			return db.Order(field + " " + order)
		},

		Filter: func(db *gorm.DB, cursor string) *gorm.DB {
			v, ok := decodeCursor[V](cursor)
			if !ok {
				return db
			}
			return db.Where(field+" "+op+" ?", v)
		},

		Next: func(item T) (string, error) {
			return encodeCursor(getValue(item)), nil
		},
	}
}