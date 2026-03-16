// module/post/model.go
package post

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Title       string         `gorm:"size:255;not null" json:"title"`
	Content     string         `gorm:"type:text" json:"content"`
	AuthorID    uint           `gorm:"index" json:"authorId"`
	Status      int8           `gorm:"default:1" json:"status"`
	ViewCount   uint64         `gorm:"default:0" json:"viewCount"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (*Post) TableName() string {
	return "posts"
}

func (p *Post) IsPublished() bool {
	return p.Status == 1
}