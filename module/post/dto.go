// module/post/dto.go
package post

import "time"

type CreatePostReq struct {
	Title   string `json:"title" binding:"required,max=255"`
	Content string `json:"content"`
}

type UpdatePostReq struct {
	Title   string `json:"title" binding:"max=255"`
	Content string `json:"content"`
	Status  *int8  `json:"status"`
}

type PostResp struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	AuthorID  uint      `json:"authorId"`
	Status    int8      `json:"status"`
	ViewCount uint64    `json:"viewCount"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PostListResp struct {
	List     []PostResp `json:"list"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
}