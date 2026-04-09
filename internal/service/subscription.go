package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"subscription-service/internal/model"
	"subscription-service/internal/repository"
)

var dateRegex = regexp.MustCompile(`^(0[1-9]|1[0-2])-\d{4}$`)

type SubscriptionService struct {
	repo   *repository.SubscriptionRepository
	logger *zap.Logger
}

func NewSubscriptionService(repo *repository.SubscriptionRepository, logger *zap.Logger) *SubscriptionService {
	return &SubscriptionService{repo: repo, logger: logger}
}

func (s *SubscriptionService) Create(ctx context.Context, req *model.CreateSubscriptionRequest) (*model.Subscription, error) {
	if req.ServiceName == "" {
		return nil, fmt.Errorf("service_name is required")
	}
	if req.Price <= 0 {
		return nil, fmt.Errorf("price must be a positive integer")
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

func (s *SubscriptionService) List(ctx context.Context) ([]model.Subscription, error) {
	return s.repo.List(ctx)
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
	if req.Price != nil && *req.Price <= 0 {
		return nil, fmt.Errorf("price must be a positive integer")
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
	if userID != "" {
		if _, err := uuid.Parse(userID); err != nil {
			return 0, fmt.Errorf("user_id must be a valid UUID")
		}
	}

	return s.repo.TotalCost(ctx, startDate, endDate, userID, serviceName)
}
