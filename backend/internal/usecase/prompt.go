package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// Prompt: super admin edit system prompt tanpa deploy
type Prompt struct {
	prompts domain.PromptRepository
}

func NewPrompt(prompts domain.PromptRepository) *Prompt {
	return &Prompt{prompts: prompts}
}

// baca prompt; fallback ke default hardcoded kalau belum di-set
func (p *Prompt) Get(ctx context.Context, key string) (*domain.Prompt, error) {
	pr, err := p.prompts.Get(ctx, key)
	if err == domain.ErrNotFound && key == domain.PromptSystem {
		return &domain.Prompt{Key: key, Content: systemPrompt}, nil
	}
	return pr, err
}

func (p *Prompt) Set(ctx context.Context, key, content string, updatedBy uuid.UUID) error {
	if strings.TrimSpace(content) == "" {
		return domain.ErrInvalidUpload
	}
	return p.prompts.Set(ctx, key, content, updatedBy)
}
