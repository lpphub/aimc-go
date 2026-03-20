// modules/post/service.go
package post

import (
	"aimc-go/app/shared/errs"
	"context"
	"time"

	"github.com/jinzhu/copier"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uint, req CreatePostReq) (*PostResp, error) {
	post := &Post{
		Title:     req.Title,
		Content:   req.Content,
		AuthorID:  userID,
		Status:    1,
		ViewCount: 0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, post); err != nil {
		return nil, err
	}

	return toResp(post), nil
}

func (s *Service) List(ctx context.Context, page, pageSize int) ([]PostResp, int64, error) {
	posts, total, err := s.repo.FindAll(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	var resp []PostResp
	for _, post := range posts {
		resp = append(resp, *toResp(&post))
	}

	return resp, total, nil
}

func (s *Service) Get(ctx context.Context, id uint) (*PostResp, error) {
	post, err := s.repo.First(ctx, id)
	if err != nil {
		return nil, errs.ErrPostNotFound
	}

	return toResp(post), nil
}

func (s *Service) Update(ctx context.Context, id uint, req UpdatePostReq) error {
	_, err := s.repo.First(ctx, id)
	if err != nil {
		return errs.ErrPostNotFound
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	return s.repo.Update(ctx, id, updates)
}

func (s *Service) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

func toResp(post *Post) *PostResp {
	var resp PostResp
	_ = copier.Copy(&resp, post)
	return &resp
}
