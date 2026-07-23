package handler

import (
	"context"

	"github.com/lexora/backend/internal/delivery/http/dto"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
)

func errBody(code, msg string) dto.Error {
	return dto.Error{Error: struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Code: code, Message: msg}}
}

// POST /auth/login
func (a *API) Login(ctx context.Context, req dto.LoginRequestObject) (dto.LoginResponseObject, error) {
	if req.Body == nil {
		return dto.Login401JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("invalid_request", "email atau password salah"))}, nil
	}
	tok, err := a.auth.Login(ctx, string(req.Body.Email), req.Body.Password)
	if err != nil {
		if err == domain.ErrRateLimited {
			return dto.Login429JSONResponse(errBody("rate_limited", "terlalu banyak percobaan login, coba lagi nanti")), nil
		}
		// generic - no enumeration
		return dto.Login401JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("invalid_credentials", "email atau password salah"))}, nil
	}
	cookie := a.cookie(tok.Refresh, a.refreshTTL)
	mcp := tok.MustChangePassword
	return dto.Login200JSONResponse{
		Body: dto.TokenResponse{
			AccessToken:        tok.Access,
			TokenType:          "Bearer",
			ExpiresIn:          tok.ExpiresIn,
			MustChangePassword: &mcp,
		},
		Headers: dto.Login200ResponseHeaders{SetCookie: &cookie},
	}, nil
}

// POST /auth/refresh
func (a *API) Refresh(ctx context.Context, _ dto.RefreshRequestObject) (dto.RefreshResponseObject, error) {
	raw := middleware.RefreshFrom(ctx)
	tok, err := a.auth.Refresh(ctx, raw)
	if err != nil {
		return dto.Refresh401JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("unauthorized", "sesi berakhir"))}, nil
	}
	cookie := a.cookie(tok.Refresh, a.refreshTTL)
	mcp := tok.MustChangePassword
	return dto.Refresh200JSONResponse{
		Body: dto.TokenResponse{
			AccessToken:        tok.Access,
			TokenType:          "Bearer",
			ExpiresIn:          tok.ExpiresIn,
			MustChangePassword: &mcp,
		},
		Headers: dto.Refresh200ResponseHeaders{SetCookie: &cookie},
	}, nil
}

// POST /auth/logout
func (a *API) Logout(ctx context.Context, _ dto.LogoutRequestObject) (dto.LogoutResponseObject, error) {
	_ = a.auth.Logout(ctx, middleware.RefreshFrom(ctx))
	clear := a.clearCookie()
	return dto.Logout204Response{Headers: dto.Logout204ResponseHeaders{SetCookie: &clear}}, nil
}

// POST /auth/change-password
func (a *API) ChangePassword(ctx context.Context, req dto.ChangePasswordRequestObject) (dto.ChangePasswordResponseObject, error) {
	id, ok := middleware.IdentityFrom(ctx)
	if !ok {
		return dto.ChangePassword401JSONResponse(errBody("unauthorized", "harus login")), nil
	}
	if req.Body == nil || len(req.Body.NewPassword) < 8 {
		return dto.ChangePassword400JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("invalid_request", "password baru minimal 8 karakter"))}, nil
	}
	if err := a.auth.ChangePassword(ctx, id.UserID, req.Body.CurrentPassword, req.Body.NewPassword); err != nil {
		if err == domain.ErrWrongPassword {
			return dto.ChangePassword400JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("wrong_password", "password saat ini salah"))}, nil
		}
		return nil, err
	}
	return dto.ChangePassword204Response{}, nil
}
