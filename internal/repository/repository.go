package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/dbpg"

	"L3.6/internal/model"
)

type postgresRepo struct {
	db      *dbpg.DB
	builder squirrel.StatementBuilderType
}

func New(dsn string) (Repository, error) {
	db, err := dbpg.New(dsn, nil, &dbpg.Options{
		MaxOpenConns:    25,
		MaxIdleConns:    25,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}
	return &postgresRepo{
		db:      db,
		builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}, nil
}

func (r *postgresRepo) Create(ctx context.Context, tx *model.Transaction) error {
	tx.ID = uuid.New().String()
	tx.CreatedAt = time.Now()
	tx.UpdatedAt = tx.CreatedAt

	query, args, err := r.builder.Insert("transactions").
		Columns("id", "type", "category", "amount", "description", "date", "created_at", "updated_at").
		Values(tx.ID, tx.Type, tx.Category, tx.Amount, tx.Description, tx.Date, tx.CreatedAt, tx.UpdatedAt).
		ToSql()
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *postgresRepo) Update(ctx context.Context, tx *model.Transaction) error {
	tx.UpdatedAt = time.Now()
	query, args, err := r.builder.Update("transactions").
		Set("type", tx.Type).
		Set("category", tx.Category).
		Set("amount", tx.Amount).
		Set("description", tx.Description).
		Set("date", tx.Date).
		Set("updated_at", tx.UpdatedAt).
		Where(squirrel.Eq{"id": tx.ID}).
		ToSql()
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *postgresRepo) Delete(ctx context.Context, id string) error {
	query, args, err := r.builder.Delete("transactions").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*model.Transaction, error) {
	query, args, err := r.builder.Select("*").
		From("transactions").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, err
	}
	var tx model.Transaction
	var date time.Time
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&tx.ID, &tx.Type, &tx.Category, &tx.Amount, &tx.Description,
		&date, &tx.CreatedAt, &tx.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tx.Date = date
	return &tx, nil
}

func (r *postgresRepo) List(ctx context.Context, filter ListFilter) ([]*model.Transaction, error) {
	b := r.builder.Select("*").From("transactions")
	if filter.From != nil {
		b = b.Where(squirrel.GtOrEq{"date": *filter.From})
	}
	if filter.To != nil {
		b = b.Where(squirrel.LtOrEq{"date": *filter.To})
	}
	if filter.Category != nil {
		b = b.Where(squirrel.Eq{"category": *filter.Category})
	}
	if filter.Type != nil {
		b = b.Where(squirrel.Eq{"type": *filter.Type})
	}
	sortBy := "date"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	order := "DESC"
	if strings.ToLower(filter.Order) == "asc" {
		order = "ASC"
	}
	b = b.OrderBy(fmt.Sprintf("%s %s", sortBy, order))
	if filter.Limit > 0 {
		b = b.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		b = b.Offset(uint64(filter.Offset))
	}
	query, args, err := b.ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*model.Transaction
	for rows.Next() {
		var tx model.Transaction
		var date time.Time
		err := rows.Scan(
			&tx.ID, &tx.Type, &tx.Category, &tx.Amount, &tx.Description,
			&date, &tx.CreatedAt, &tx.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tx.Date = date
		transactions = append(transactions, &tx)
	}
	return transactions, rows.Err()
}

func (r *postgresRepo) GetAnalytics(ctx context.Context, from, to time.Time) (*AnalyticsResult, error) {
	query := `
		SELECT 
			COALESCE(SUM(amount), 0) as sum,
			COALESCE(AVG(amount), 0) as avg,
			COUNT(*) as count,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY amount), 0) as median,
			COALESCE(PERCENTILE_CONT(0.9) WITHIN GROUP (ORDER BY amount), 0) as p90
		FROM transactions
		WHERE date >= $1 AND date <= $2
	`
	var res AnalyticsResult
	err := r.db.QueryRowContext(ctx, query, from, to).Scan(
		&res.Sum, &res.Avg, &res.Count, &res.Median, &res.Percentile90,
	)
	return &res, err
}

func (r *postgresRepo) GetGroupedAnalytics(ctx context.Context, from, to time.Time, groupBy string) ([]*GroupedResult, error) {
	var groupExpr string
	switch groupBy {
	case "day":
		groupExpr = "date::date"
	case "week":
		groupExpr = "DATE_TRUNC('week', date)::date"
	case "category":
		groupExpr = "category"
	default:
		return nil, fmt.Errorf("unsupported group_by: %s", groupBy)
	}
	query := fmt.Sprintf(`
		SELECT 
			%s as group_key,
			COALESCE(SUM(amount), 0) as sum,
			COALESCE(AVG(amount), 0) as avg,
			COUNT(*) as count,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY amount), 0) as median,
			COALESCE(PERCENTILE_CONT(0.9) WITHIN GROUP (ORDER BY amount), 0) as p90
		FROM transactions
		WHERE date >= $1 AND date <= $2
		GROUP BY group_key
		ORDER BY group_key
	`, groupExpr)
	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*GroupedResult
	for rows.Next() {
		var gr GroupedResult
		err := rows.Scan(&gr.GroupKey, &gr.Sum, &gr.Avg, &gr.Count, &gr.Median, &gr.Pct90)
		if err != nil {
			return nil, err
		}
		results = append(results, &gr)
	}
	return results, rows.Err()
}

func (r *postgresRepo) ListForExport(ctx context.Context, from, to time.Time) ([]*model.Transaction, error) {
	query, args, err := r.builder.Select("*").
		From("transactions").
		Where(squirrel.And{
			squirrel.GtOrEq{"date": from},
			squirrel.LtOrEq{"date": to},
		}).
		OrderBy("date DESC").
		ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var transactions []*model.Transaction
	for rows.Next() {
		var tx model.Transaction
		var date time.Time
		err := rows.Scan(
			&tx.ID, &tx.Type, &tx.Category, &tx.Amount, &tx.Description,
			&date, &tx.CreatedAt, &tx.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tx.Date = date
		transactions = append(transactions, &tx)
	}
	return transactions, rows.Err()
}

func (r *postgresRepo) Ping(ctx context.Context) error {
	return r.db.Master.PingContext(ctx)
}

func (r *postgresRepo) Close() error {
	return r.db.Master.Close()
}
