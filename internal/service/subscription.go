package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"subscription-service/internal/model"
)

var dateRegex = regexp.MustCompile(`^(0[1-9]|1[0-2])-\d{4}$`)

// Repository defines the interface for subscription data access.
type Repository interface {
	Create(ctx context.Context, sub *model.Subscription) (*model.Subscription, error)
	GetByID(ctx context.Context, id int) (*model.Subscription, error)
	List(ctx context.Context, userID, serviceName string) ([]model.Subscription, error)
	Update(ctx context.Context, id int, req *model.UpdateSubscriptionRequest) (*model.Subscription, error)
	Delete(ctx context.Context, id int) error
}

type SubscriptionService struct {
	repo   Repository
	logger *zap.Logger
}

func NewSubscriptionService(repo Repository, logger *zap.Logger) *SubscriptionService {
	return &SubscriptionService{repo: repo, logger: logger}
}

func (s *SubscriptionService) Create(ctx context.Context, req *model.CreateSubscriptionRequest) (*model.Subscription, error) {
	if req.ServiceName == "" {
		return nil, fmt.Errorf("service_name is required")
	}
	if req.Price < 0 {
		return nil, fmt.Errorf("price must be a non-negative integer")
	}
	if !dateRegex.MatchString(req.StartDate) {
		return nil, fmt.Errorf("start_date must be in MM-YYYY format")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user_id must be a valid UUID")
	}

	var endDate *string
	if req.EndDate != "" {
		if !dateRegex.MatchString(req.EndDate) {
			return nil, fmt.Errorf("end_date must be in MM-YYYY format")
		}
		before, err := isDateBefore(req.EndDate, req.StartDate)
		if err != nil {
			return nil, err
		}
		if before {
			return nil, fmt.Errorf("end_date must not be before start_date")
		}
		endDate = &req.EndDate
	}

	sub := &model.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      userID,
		StartDate:   req.StartDate,
		EndDate:     endDate,
	}

	s.logger.Info("creating subscription via service layer",
		zap.String("service_name", req.ServiceName),
		zap.String("user_id", req.UserID),
	)

	return s.repo.Create(ctx, sub)
}

func (s *SubscriptionService) GetByID(ctx context.Context, id int) (*model.Subscription, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id must be a positive integer")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *SubscriptionService) List(ctx context.Context, userID, serviceName string) ([]model.Subscription, error) {
	return s.repo.List(ctx, userID, serviceName)
}

func (s *SubscriptionService) Update(ctx context.Context, id int, req *model.UpdateSubscriptionRequest) (*model.Subscription, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id must be a positive integer")
	}

	if req.StartDate != nil && !dateRegex.MatchString(*req.StartDate) {
		return nil, fmt.Errorf("start_date must be in MM-YYYY format")
	}
	if req.EndDate != nil && !dateRegex.MatchString(*req.EndDate) {
		return nil, fmt.Errorf("end_date must be in MM-YYYY format")
	}
	if req.Price != nil && *req.Price < 0 {
		return nil, fmt.Errorf("price must be a non-negative integer")
	}
	if req.StartDate != nil && req.EndDate != nil {
		before, err := isDateBefore(*req.EndDate, *req.StartDate)
		if err != nil {
			return nil, err
		}
		if before {
			return nil, fmt.Errorf("end_date must not be before start_date")
		}
	}

	return s.repo.Update(ctx, id, req)
}

func (s *SubscriptionService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("id must be a positive integer")
	}
	return s.repo.Delete(ctx, id)
}

func (s *SubscriptionService) TotalCost(ctx context.Context, startDate, endDate, userID, serviceName string) (int, error) {
	if !dateRegex.MatchString(startDate) {
		return 0, fmt.Errorf("start_date must be in MM-YYYY format")
	}
	if !dateRegex.MatchString(endDate) {
		return 0, fmt.Errorf("end_date must be in MM-YYYY format")
	}

	before, err := isDateBefore(endDate, startDate)
	if err != nil {
		return 0, err
	}
	if before {
		return 0, fmt.Errorf("end_date must not be before start_date")
	}

	if userID != "" {
		if _, err := uuid.Parse(userID); err != nil {
			return 0, fmt.Errorf("user_id must be a valid UUID")
		}
	}

	subs, err := s.repo.List(ctx, userID, serviceName)
	if err != nil {
		return 0, err
	}

	var total int
	for _, sub := range subs {
		months, err := overlapMonths(sub.StartDate, sub.EndDate, startDate, endDate)
		if err != nil {
			s.logger.Error("failed to calculate overlap", zap.Error(err))
			continue
		}
		total += sub.Price * months
	}

	return total, nil
}
