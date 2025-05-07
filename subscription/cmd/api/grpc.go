package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ziliscite/bard_narate/subscription/internal/domain"
	"github.com/ziliscite/bard_narate/subscription/internal/repository"
	"github.com/ziliscite/bard_narate/subscription/internal/service"
	"github.com/ziliscite/bard_narate/subscription/pkg/midtrans"
	pb "github.com/ziliscite/bard_narate/subscription/pkg/protobuf"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type App struct {
	so  service.Order
	sp  service.Payment
	spr service.Product
	pb.UnimplementedOrderServiceServer
}

func NewApp(so service.Order, po service.Payment, pro service.Product) App {
	return App{
		so:  so,
		sp:  po,
		spr: pro,
	}
}

func (a *App) Checkout(ctx context.Context, req *pb.CheckoutRequest) (*pb.CheckoutResponse, error) {
	plan, err := a.spr.GetPlan(ctx, req.GetPlanId())
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return nil, status.Error(codes.NotFound, "plan not found")
		default:
			return nil, status.Error(codes.Internal, "failed to get plan")
		}
	}

	discount, err := a.spr.GetDiscount(ctx, req.GetDiscountCode(), plan.ID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmptyDiscount):
			// nothing
		case errors.Is(err, domain.ErrInvalidDiscount) || errors.Is(err, domain.ErrExpiredDiscount) || errors.Is(err, domain.ErrInActiveDiscount):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, "failed to get discount")
		}
	}

	args := make([]domain.ProcessingOption, 0)
	if discount != nil {
		args = append(args, domain.WithDiscount(discount.PercentageValue))
	}
	args = append(args, domain.WithFees(2), domain.WithTax(12))

	transaction, err := a.so.Checkout(ctx, req.UserId, plan, args...)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to checkout")
	}

	url, err := a.sp.GetSnapURL(ctx, transaction, plan)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to checkout")
	}

	return &pb.CheckoutResponse{
		PaymentUrl: url,
	}, nil
}

func (a *App) HandleWebhook(ctx context.Context, req *pb.WebhookRequest) (*pb.WebhookResponse, error) {
	var notification midtrans.PaymentStatus
	if err := json.Unmarshal(req.GetPayload(), &notification); err != nil {
		return nil, err
	}

	ps, err := a.sp.VerifyPayment(ctx, &notification)
	if err != nil {
		return nil, err
	}

	switch ps {
	case domain.Pending:
		return &pb.WebhookResponse{Status: pb.Status_Pending}, nil
	case domain.Completed:
		return a.approve(ctx, notification.OrderId)
	case domain.Failed:
		return a.cancel(ctx, notification.OrderId)
	default:
		return nil, status.Error(codes.Internal, "failed to checkout")
	}
}

func (a *App) approve(ctx context.Context, transactionID string) (*pb.WebhookResponse, error) {
	if err := a.so.Finalize(ctx, transactionID); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return nil, status.Error(codes.NotFound, "transaction not found")
		default:
			return nil, status.Error(codes.Internal, "failed to approve transaction")
		}
	}

	return &pb.WebhookResponse{Status: pb.Status_Completed}, nil
}

func (a *App) cancel(ctx context.Context, transactionID string) (*pb.WebhookResponse, error) {
	if err := a.so.Cancel(ctx, transactionID); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return nil, status.Error(codes.NotFound, "transaction not found")
		default:
			return nil, status.Error(codes.Internal, "failed to cancel transaction")
		}
	}

	return &pb.WebhookResponse{Status: pb.Status_Failed}, nil
}
