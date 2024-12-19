package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/michgboxy2/carzone/models"
)

type CarStoreInterface interface {
	GetCarById(ctx context.Context, id string) (models.Car, error)
	GetCarByBrand(ctx context.Context, brand string, isEngine bool) ([]models.Car, error)
	CreateCar(ctx context.Context, carReq *models.CarRequest) (models.Car, error)
	UpdateCar(ctx context.Context, id uuid.UUID, carReq *models.CarRequest) (models.Car, error)
	DeleteCar(ctx context.Context, id string) (models.Car, error)
}

type EngineStoreInterface interface {
	GetEngineById(ctx context.Context, id string) (models.Engine, error)
	CreateEngine(ctx context.Context, engineReq *models.EngineRequest) (models.Engine, error)
	EngineUpdate(ctx context.Context, id uuid.UUID, engineReq *models.EngineRequest) (models.Engine, error)
	EngineDelete(ctx context.Context, id string) (models.Engine, error)
}
