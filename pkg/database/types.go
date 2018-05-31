package database

import (
	"context"
	"io"

	"git.containerum.net/ch/volume-manager/pkg/models"
)

type DB interface {
	StorageByName(ctx context.Context, name string) (model.Storage, error)
	LeastUsedStorage(ctx context.Context, requestSize int) (model.Storage, error)
	AllStorages(ctx context.Context) ([]model.Storage, error)
	CreateStorage(ctx context.Context, storage *model.Storage) error
	UpdateStorage(ctx context.Context, name string, storage model.Storage) error
	DeleteStorage(ctx context.Context, storage *model.Storage) error

	VolumeByID(ctx context.Context, userID, label string) (model.Volume, error)
	UserVolumes(ctx context.Context, userID string, filter VolumeFilter) ([]model.Volume, error)
	AllVolumes(ctx context.Context, filter VolumeFilter) ([]model.Volume, error)
	CreateVolume(ctx context.Context, volume *model.Volume) error
	DeleteVolume(ctx context.Context, volume *model.Volume) error
	DeleteAllUserVolumes(ctx context.Context, userID string) ([]model.Volume, error)
	UpdateVolume(ctx context.Context, volume *model.Volume) error

	Transactional(func(tx DB) error) error
	io.Closer
}
