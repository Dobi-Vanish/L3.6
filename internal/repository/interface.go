package repository

import (
	"context"
	"time"

	"L3.6/internal/model"
)

type Repository interface {
	Create(ctx context.Context, tx *model.Transaction) error
	Update(ctx context.Context, tx *model.Transaction) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*model.Transaction, error)
	List(ctx context.Context, filter ListFilter) ([]*model.Transaction, error)
	GetAnalytics(ctx context.Context, from, to time.Time) (*AnalyticsResult, error)
	GetGroupedAnalytics(ctx context.Context, from, to time.Time, groupBy string) ([]*GroupedResult, error)
	ListForExport(ctx context.Context, from, to time.Time) ([]*model.Transaction, error)
	Ping(ctx context.Context) error
	Close() error
}

type ListFilter struct {
	From     *time.Time
	To       *time.Time
	Category *string
	Type     *string
	SortBy   string
	Order    string
	Limit    int
	Offset   int
}

type AnalyticsResult struct {
	Sum          float64 `json:"sum"`
	Avg          float64 `json:"avg"`
	Count        int     `json:"count"`
	Median       float64 `json:"median"`
	Percentile90 float64 `json:"percentile_90"`
}

type GroupedResult struct {
	GroupKey string  `json:"group_key"`
	Sum      float64 `json:"sum"`
	Avg      float64 `json:"avg"`
	Count    int     `json:"count"`
	Median   float64 `json:"median"`
	Pct90    float64 `json:"pct90"`
}
