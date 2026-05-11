// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"fmt"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

type recordsService struct {
	repo domain.RecordRepository
}

// NewRecordsService returns a RecordService backed by the given repository.
func NewRecordsService(repo domain.RecordRepository) domain.RecordService {
	return &recordsService{repo: repo}
}

func (s *recordsService) Create(ctx context.Context, r domain.Record) (domain.Record, error) {
	if r.SourceID == "" {
		return domain.Record{}, fmt.Errorf("source_id is required")
	}
	if r.Status == "" {
		r.Status = domain.RecordStatusQueued
	}
	return s.repo.Create(ctx, r)
}

func (s *recordsService) GetByID(ctx context.Context, id, userID string) (domain.Record, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *recordsService) List(
	ctx context.Context,
	userID string,
	f domain.RecordFilter,
	p domain.Page,
) (domain.RecordPage, error) {
	return s.repo.List(ctx, userID, f, p)
}

func (s *recordsService) Delete(ctx context.Context, id, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *recordsService) RetryIngest(ctx context.Context, id, userID string) error {
	if _, err := s.repo.GetByID(ctx, id, userID); err != nil {
		return err
	}
	return s.repo.UpdateStatus(ctx, id, domain.RecordStatusQueued, "")
}
