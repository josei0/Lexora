package handler

import (
	"context"

	"github.com/lexora/backend/internal/delivery/http/dto"
)

// implements dto.StrictServerInterface
type API struct{}

func New() *API {
	return &API{}
}

// liveness
func (a *API) GetHealth(ctx context.Context, _ dto.GetHealthRequestObject) (dto.GetHealthResponseObject, error) {
	return dto.GetHealth200JSONResponse{Status: "ok"}, nil
}
