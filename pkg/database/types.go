package database

import (
	"context"
	"io"

	"git.containerum.net/ch/volume-manager/pkg/models"
)

type DB interface {
	Storages(ctx context.Context) ([]model.Storage, error)
	CreateStorage(ctx context.Context, storage *model.Storage) error
	UpdateStorage(ctx context.Context, storage model.Storage) error
	DeleteStorage(ctx context.Context, storage model.Storage) error

	VolumeByLabel(ctx context.Context, userID, label string) (model.Volume, error)
	UserVolumes(ctx context.Context, userID string) ([]model.Volume, error)
	AllVolumes(ctx context.Context) ([]model.Volume, error)
	CreateVolume(ctx context.Context, volume *model.Volume) error
	DeleteVolume(ctx context.Context, volume model.Volume) error
	UpdateVolume(ctx context.Context, volume model.Volume) error

	Transactional(func(tx DB) error) error
	io.Closer
}
