package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, in CreateSubscriptionInput) (Subscription, error) {
	if in.ServiceName == "" {
		return Subscription{}, fmt.Errorf("service_name is required")
	}
	if in.Price <= 0 {
		return Subscription{}, fmt.Errorf("price must be a positive integer")
	}
	if in.UserID == "" {
		return Subscription{}, fmt.Errorf("user_id is required")
	}
	if in.StartDate == "" {
		return Subscription{}, fmt.Errorf("start_date is required")
	}

	sub, err := s.repo.Create(ctx, in)
	if err != nil {
		slog.Error("create subscription failed", "error", err)
		return Subscription{}, err
	}
	slog.Info("subscription created", "id", sub.ID, "user_id", sub.UserID, "service", sub.ServiceName)
	return sub, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (Subscription, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			slog.Error("get subscription failed", "id", id, "error", err)
		}
		return Subscription{}, err
	}
	return sub, nil
}

func (s *Service) List(ctx context.Context, userID, serviceName string) ([]Subscription, error) {
	subs, err := s.repo.List(ctx, userID, serviceName)
	if err != nil {
		slog.Error("list subscriptions failed", "error", err)
		return nil, err
	}
	return subs, nil
}

func (s *Service) Update(ctx context.Context, id string, in UpdateSubscriptionInput) (Subscription, error) {
	if in.Price != nil && *in.Price <= 0 {
		return Subscription{}, fmt.Errorf("price must be a positive integer")
	}

	sub, err := s.repo.Update(ctx, id, in)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			slog.Error("update subscription failed", "id", id, "error", err)
		}
		return Subscription{}, err
	}
	slog.Info("subscription updated", "id", id)
	return sub, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if !errors.Is(err, ErrNotFound) {
			slog.Error("delete subscription failed", "id", id, "error", err)
		}
		return err
	}
	slog.Info("subscription deleted", "id", id)
	return nil
}

func (s *Service) TotalCost(ctx context.Context, f TotalCostFilter) (int, error) {
	if f.From == "" {
		return 0, fmt.Errorf("from is required (format: MM-YYYY)")
	}
	if f.To == "" {
		return 0, fmt.Errorf("to is required (format: MM-YYYY)")
	}

	total, err := s.repo.TotalCost(ctx, f)
	if err != nil {
		slog.Error("total cost calculation failed", "filter", f, "error", err)
		return 0, err
	}
	slog.Info("total cost calculated", "filter", f, "total", total)
	return total, nil
}
