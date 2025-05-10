package main

import (
	"context"
	"github.com/ziliscite/bard_narate/auth/internal/service"
	pb "github.com/ziliscite/bard_narate/auth/pkg/protobuf"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthenticationServer struct {
	as  service.ServerAuthenticator
	oas service.OAuthAuthenticator

	pb.UnimplementedOAuthServiceServer
	pb.UnimplementedServerAuthServiceServer
}

func NewAuthenticationService(as service.ServerAuthenticator, oas service.OAuthAuthenticator) *AuthenticationServer {
	return &AuthenticationServer{
		as:  as,
		oas: oas,
	}
}

func (s *AuthenticationServer) AuthenticationURL(_ context.Context, req *pb.URLRequest) (*pb.URLResponse, error) {
	provider := req.GetProvider().String()
	url, err := s.oas.AuthenticationURL(provider, req.GetState())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get authentication URL")
	}

	return &pb.URLResponse{
		Url: url,
	}, nil
}

func (s *AuthenticationServer) AuthenticationCallback(ctx context.Context, req *pb.CallbackRequest) (*pb.CallbackResponse, error) {
	token, err := s.oas.AuthenticationCallback(ctx, req.Provider.String(), req.GetCode())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to exchange code")
	}

	return &pb.CallbackResponse{
		AccessToken:          token.AccessToken,
		RefreshToken:         token.RefreshToken,
		AccessTokenExpireAt:  timestamppb.New(token.AccessTokenExpiresAt),
		RefreshTokenExpireAt: timestamppb.New(token.RefreshTokenExpiresAt),
	}, nil
}

func (s *AuthenticationServer) UserInfo(ctx context.Context, req *pb.UserInfoRequest) (*pb.UserInfoResponse, error) {
	user, err := s.as.Authenticate(ctx, req.GetAccessToken())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user info")
	}

	var provider pb.Provider
	switch user.Provider {
	case "GITHUB":
		provider = pb.Provider_GITHUB
	case "GOOGLE":
		provider = pb.Provider_GOOGLE
	default:
		// should panic, because we should not have an unknown provider
		return nil, status.Error(codes.InvalidArgument, "invalid provider")
	}

	return &pb.UserInfoResponse{
		Id:             user.ID,
		Provider:       provider,
		ProviderUserId: user.ProviderUserID,
		Picture:        user.Picture,
		Email:          user.Email,
		Name:           user.Name,
		Username:       user.Username,
	}, nil
}

func (s *AuthenticationServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	token, err := s.as.Refresh(ctx, req.GetAccessToken(), req.GetRefreshToken())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to refresh token")
	}

	return &pb.RefreshTokenResponse{
		AccessToken:          token.AccessToken,
		RefreshToken:         token.RefreshToken,
		AccessTokenExpireAt:  timestamppb.New(token.AccessTokenExpiresAt),
		RefreshTokenExpireAt: timestamppb.New(token.RefreshTokenExpiresAt),
	}, nil
}
