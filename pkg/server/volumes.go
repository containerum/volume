package server

import (
	"context"

	"git.containerum.net/ch/volume-manager/pkg/database"
	"git.containerum.net/ch/volume-manager/pkg/models"
	billing "github.com/containerum/bill-external/models"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/sirupsen/logrus"
)

type VolumeActions interface {
	AdminCreateVolume(ctx context.Context, nsID string, req model.AdminVolumeCreateRequest) error
	CreateVolume(ctx context.Context, nsID string, req model.VolumeCreateRequest) error
	RenameVolume(ctx context.Context, id, newLabel string) error
	ResizeVolume(ctx context.Context, id string, newTariffID string) error
	GetVolume(ctx context.Context, id string) (kubeClientModel.Volume, error)
	GetUserVolumes(ctx context.Context, filters ...string) ([]kubeClientModel.Volume, error)
	GetAllVolumes(ctx context.Context, page, perPage int, filters ...string) ([]kubeClientModel.Volume, error)
	DeleteVolume(ctx context.Context, id string) error
	DeleteAllUserVolumes(ctx context.Context) error
}

var StandardVolumeFilter = database.VolumeFilter{
	NotDeleted: true,
}

func (s *Server) AdminCreateVolume(ctx context.Context, nsID string, req model.AdminVolumeCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"ns_id":    nsID,
		"capacity": req.Capacity,
		"label":    req.Label,
		"user_id":  userID,
	}).Infof("create volume")

	err := s.db.Transactional(func(tx database.DB) error {
		storage, getErr := tx.LeastUsedStorage(ctx, req.Capacity)
		if getErr != nil {
			return getErr
		}

		volume := model.Volume{
			Resource: model.Resource{
				Label: req.Label,
			},
			Capacity:    req.Capacity,
			NamespaceID: nsID,
			StorageID:   storage.ID,
		}

		if createErr := tx.CreateVolume(ctx, &volume); createErr != nil {
			return createErr
		}

		// TODO: create it actually

		return nil
	})

	return err
}

func (s *Server) CreateVolume(ctx context.Context, nsID string, req model.VolumeCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"ns_id":     nsID,
		"tariff_id": req.TariffID,
		"id":        req.Label,
		"user_id":   userID,
	}).Infof("create volume")

	tariff, err := s.clients.Billing.GetVolumeTariff(ctx, req.TariffID)
	if err != nil {
		return err
	}

	if chkErr := CheckTariff(tariff.Tariff, IsAdminRole(ctx)); chkErr != nil {
		return chkErr
	}

	err = s.db.Transactional(func(tx database.DB) error {
		storage, getErr := tx.LeastUsedStorage(ctx, tariff.StorageLimit)
		if getErr != nil {
			return getErr
		}

		volume := model.Volume{
			Resource: model.Resource{
				TariffID: &tariff.ID,
				Label:    req.Label,
			},
			Capacity:    tariff.StorageLimit,
			NamespaceID: nsID,
			StorageID:   storage.ID,
		}

		if createErr := tx.CreateVolume(ctx, &volume); createErr != nil {
			return createErr
		}

		// TODO: create it actually

		subReq := billing.SubscribeTariffRequest{
			TariffID:      tariff.ID,
			ResourceType:  billing.Volume,
			ResourceLabel: volume.Label,
			ResourceID:    volume.ID,
		}
		if subErr := s.clients.Billing.Subscribe(ctx, subReq); subErr != nil {
			return subErr
		}

		return nil
	})

	return err
}

func (s *Server) GetVolume(ctx context.Context, id string) (kubeClientModel.Volume, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
	}).Infof("get volume")

	vol, err := s.db.VolumeByID(ctx, id)
	if err != nil {
		return vol.ToKube(), err
	}

	return vol.ToKube(), nil
}

func (s *Server) GetUserVolumes(ctx context.Context, filters ...string) ([]kubeClientModel.Volume, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"filters": filters,
	}).Infof("get user volumes")

	var filter database.VolumeFilter
	if IsAdminRole(ctx) {
		filter = database.ParseVolumeFilter(filters...)
	} else {
		filter = StandardVolumeFilter
	}

	vols, err := s.db.VolumesByIDs(ctx, nil /* TODO */, filter)
	if err != nil {
		return nil, err
	}

	ret := make([]kubeClientModel.Volume, len(vols))
	for i := range vols {
		ret[i] = vols[i].ToKube()
	}

	return ret, nil
}

func (s *Server) GetAllVolumes(ctx context.Context, page, perPage int, filters ...string) ([]kubeClientModel.Volume, error) {
	s.log.WithFields(logrus.Fields{
		"page":     page,
		"per_page": perPage,
		"filters":  filters,
	}).Infof("get all volumes")

	var filter database.VolumeFilter
	if len(filters) > 0 {
		filter = database.ParseVolumeFilter()
	} else {
		filter = StandardVolumeFilter
	}
	filter.PerPage = perPage
	filter.Page = page

	vols, err := s.db.AllVolumes(ctx, filter)
	if err != nil {
		return nil, err
	}

	ret := make([]kubeClientModel.Volume, len(vols))
	for i := range vols {
		ret[i] = vols[i].ToKube()
	}

	return ret, nil
}

func (s *Server) DeleteVolume(ctx context.Context, id string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
	}).Infof("delete volume")

	err := s.db.Transactional(func(tx database.DB) error {
		vol, getErr := tx.VolumeByID(ctx, id)
		if getErr != nil {
			return getErr
		}

		if delErr := tx.DeleteVolume(ctx, &vol); delErr != nil {
			return delErr
		}

		// TODO: actually delete it

		if unsubErr := s.clients.Billing.Unsubscribe(ctx, vol.ID); unsubErr != nil {
			return unsubErr
		}

		return nil
	})

	return err
}

func (s *Server) DeleteAllUserVolumes(ctx context.Context) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithField("user_id", userID).Infof("delete all user volumes")

	err := s.db.Transactional(func(tx database.DB) error {
		vols, err := s.db.VolumesByIDs(ctx, nil /* TODO */, StandardVolumeFilter)
		if err != nil {
			return err
		}

		if delErr := tx.DeleteVolumes(ctx, vols); delErr != nil {
			return delErr
		}

		// TODO: actually delete it

		var resourceIDs []string
		for _, v := range vols {
			resourceIDs = append(resourceIDs, v.ID)
		}
		if unsubErr := s.clients.Billing.MassiveUnsubscribe(ctx, resourceIDs); unsubErr != nil {
			return unsubErr
		}

		return nil
	})

	return err
}

func (s *Server) RenameVolume(ctx context.Context, id, newLabel string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"id":      id,
		"new_id":  newLabel,
	}).Infof("rename volume")

	err := s.db.Transactional(func(tx database.DB) error {
		vol, getErr := tx.VolumeByID(ctx, id)
		if getErr != nil {
			return getErr
		}

		vol.Label = newLabel

		if renErr := tx.UpdateVolume(ctx, &vol); renErr != nil {
			return renErr
		}

		// TODO: rename it actually

		if renErr := s.clients.Billing.Rename(ctx, vol.ID, newLabel); renErr != nil {
			return renErr
		}

		return nil
	})

	return err
}

func (s *Server) ResizeVolume(ctx context.Context, id string, newTariffID string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":       userID,
		"id":            id,
		"new_tariff_id": newTariffID,
	}).Infof("resize volume")

	newTariff, err := s.clients.Billing.GetVolumeTariff(ctx, newTariffID)
	if err != nil {
		return err
	}

	if chkErr := CheckTariff(newTariff.Tariff, IsAdminRole(ctx)); chkErr != nil {
		return chkErr
	}

	err = s.db.Transactional(func(tx database.DB) error {
		vol, getErr := tx.VolumeByID(ctx, id)
		if getErr != nil {
			return getErr
		}

		vol.TariffID = &newTariff.ID
		vol.Capacity = newTariff.StorageLimit

		if resizeErr := tx.UpdateVolume(ctx, &vol); resizeErr != nil {
			return resizeErr
		}

		// TODO: resize it actually

		return nil
	})

	return err
}
