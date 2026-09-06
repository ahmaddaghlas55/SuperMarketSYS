package services

import (
	"context"

	"supermarket/internal/models"
	"supermarket/internal/repository"
)

type AuditService struct{ repo *repository.AuditRepository }

func NewAuditService(r *repository.AuditRepository) *AuditService { return &AuditService{repo: r} }

func (s *AuditService) List(ctx context.Context, from, to, module string, userID int64) ([]models.AuditEntry, error) {
	return s.repo.List(ctx, from, to, module, userID)
}
