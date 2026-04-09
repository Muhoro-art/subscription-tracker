package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"subscription-service/internal/model"
)

type SubscriptionRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewSubscriptionRepository(db *sqlx.DB, logger *zap.Logger) *SubscriptionRepository {
	return &SubscriptionRepository{db: db, logger: logger}
}

func (r *SubscriptionRepository) Create(ctx context.Context, sub *model.Subscription) (*model.Subscription, error) {
	query := `
		INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, service_name, price, user_id, start_date, end_date, created_at, updated_at`

	r.logger.Info("creating subscription",
		zap.String("service_name", sub.ServiceName),
		zap.String("user_id", sub.UserID.String()),
	)

	var created model.Subscription
	err := r.db.QueryRowxContext(ctx, query,
		sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.EndDate,
	).StructScan(&created)
	if err != nil {
		r.logger.Error("failed to create subscription", zap.Error(err))
		return nil, fmt.Errorf("create subscription: %w", err)
	}

	return &created, nil
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, id int) (*model.Subscription, error) {
	query := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
	           FROM subscriptions WHERE id = $1`

	r.logger.Info("getting subscription by id", zap.Int("id", id))

	var sub model.Subscription
	err := r.db.GetContext(ctx, &sub, query, id)
	if err != nil {
		r.logger.Error("failed to get subscription", zap.Int("id", id), zap.Error(err))
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	return &sub, nil
}

func (r *SubscriptionRepository) List(ctx context.Context, userID, serviceName string) ([]model.Subscription, error) {
	query := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
	           FROM subscriptions WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if userID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	if serviceName != "" {
		query += fmt.Sprintf(" AND service_name = $%d", argIdx)
		args = append(args, serviceName)
	}

	query += " ORDER BY id"

	r.logger.Info("listing subscriptions",
		zap.String("user_id", userID),
		zap.String("service_name", serviceName),
	)

	var subs []model.Subscription
	err := r.db.SelectContext(ctx, &subs, query, args...)
	if err != nil {
		r.logger.Error("failed to list subscriptions", zap.Error(err))
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}

	return subs, nil
}

func (r *SubscriptionRepository) Update(ctx context.Context, id int, sub *model.UpdateSubscriptionRequest) (*model.Subscription, error) {
	query := `
		UPDATE subscriptions SET
			service_name = COALESCE($1, service_name),
			price = COALESCE($2, price),
			start_date = COALESCE($3, start_date),
			end_date = COALESCE($4, end_date),
			updated_at = NOW()
		WHERE id = $5
		RETURNING id, service_name, price, user_id, start_date, end_date, created_at, updated_at`

	r.logger.Info("updating subscription", zap.Int("id", id))

	var updated model.Subscription
	err := r.db.QueryRowxContext(ctx, query,
		sub.ServiceName, sub.Price, sub.StartDate, sub.EndDate, id,
	).StructScan(&updated)
	if err != nil {
		r.logger.Error("failed to update subscription", zap.Int("id", id), zap.Error(err))
		return nil, fmt.Errorf("update subscription: %w", err)
	}

	return &updated, nil
}

func (r *SubscriptionRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM subscriptions WHERE id = $1`

	r.logger.Info("deleting subscription", zap.Int("id", id))

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("failed to delete subscription", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("delete subscription: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete subscription rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("subscription with id %d not found", id)
	}

	return nil
}
