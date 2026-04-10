package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"L3.6/internal/model"
	"L3.6/internal/repository"
)

type TransactionService struct {
	repo repository.Repository
}

func NewTransactionService(repo repository.Repository) *TransactionService {
	return &TransactionService{repo: repo}
}

func (s *TransactionService) Create(ctx context.Context, tx *model.Transaction) error {
	if err := validateTransaction(tx); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return s.repo.Create(ctx, tx)
}

func (s *TransactionService) Update(ctx context.Context, tx *model.Transaction) error {
	if tx.ID == "" {
		return errors.New("id is required")
	}
	if err := validateTransaction(tx); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return s.repo.Update(ctx, tx)
}

func (s *TransactionService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("id is required")
	}
	return s.repo.Delete(ctx, id)
}

func (s *TransactionService) GetByID(ctx context.Context, id string) (*model.Transaction, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *TransactionService) List(ctx context.Context, filter repository.ListFilter) ([]*model.Transaction, error) {
	return s.repo.List(ctx, filter)
}

func (s *TransactionService) GetAnalytics(ctx context.Context, from, to time.Time) (*repository.AnalyticsResult, error) {
	if from.IsZero() || to.IsZero() {
		return nil, errors.New("from and to dates are required")
	}
	if from.After(to) {
		return nil, errors.New("from date must be before to date")
	}
	return s.repo.GetAnalytics(ctx, from, to)
}

func (s *TransactionService) GetGroupedAnalytics(ctx context.Context, from, to time.Time, groupBy string) ([]*repository.GroupedResult, error) {
	if from.IsZero() || to.IsZero() {
		return nil, errors.New("from and to dates are required")
	}
	if from.After(to) {
		return nil, errors.New("from date must be before to date")
	}
	return s.repo.GetGroupedAnalytics(ctx, from, to, groupBy)
}

func (s *TransactionService) ExportCSV(ctx context.Context, from, to time.Time) ([]*model.Transaction, error) {
	if from.IsZero() || to.IsZero() {
		return nil, errors.New("from and to dates are required")
	}
	if from.After(to) {
		return nil, errors.New("from date must be before to date")
	}
	return s.repo.ListForExport(ctx, from, to)
}

func validateTransaction(tx *model.Transaction) error {
	if tx.Type != "income" && tx.Type != "expense" {
		return errors.New("type must be 'income' or 'expense'")
	}
	if tx.Category == "" {
		return errors.New("category is required")
	}
	if tx.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if tx.Date.IsZero() {
		return errors.New("date is required")
	}
	if tx.Date.After(time.Now()) {
		return errors.New("date cannot be in the future")
	}
	return nil
}
