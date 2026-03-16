// module/post/repository.go
package post

import (
	"context"

	"github.com/lpphub/goweb/ext/dbx"
	"gorm.io/gorm"
)

type Repository struct {
	*dbx.BaseRepo[Post]
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		BaseRepo: dbx.NewBaseRepo[Post](db),
	}
}

func (r *Repository) FindAll(ctx context.Context, page, pageSize int) ([]Post, int64, error) {
	var posts []Post
	var total int64

	db := r.DB().WithContext(ctx).Model(&Post{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := db.Order("id DESC").Offset(offset).Limit(pageSize).Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}