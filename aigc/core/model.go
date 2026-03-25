package core

import "context"

type Model interface {
	ID() ModelID
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
}
