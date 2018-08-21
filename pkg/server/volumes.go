package server

import (
	"context"

	"git.containerum.net/ch/volume-manager/pkg/database"
	"git.containerum.net/ch/volume-manager/pkg/errors"
	"git.containerum.net/ch/volume-manager/pkg/models"
	billing "github.com/containerum/bill-external/models"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

var (
	_ VolumeActions = new(Server)
)

type VolumeActions interface {
	DirectCreateVolume(ctx context.Context, nsID string, req model.DirectVolumeCreateRequest) error
	CreateVolume(ctx context.Context, nsID string, req model.VolumeCreateRequest) error
	ImportVolume(ctx context.Context, nsID string, req kubeClientModel.Volume) error
	AdminResizeVolume(ctx context.Context, nsID, label string, newCapacity int) error
	ResizeVolume(ctx context.Context, nsID, label string, newTariffID string) error
	GetVolume(ctx context.Context, nsID, label string) (kubeClientModel.Volume, error)
	GetUserVolumes(ctx context.Context) (kubeClientModel.VolumesList, error)
	GetNamespaceVolumes(ctx context.Context, nsID string) (kubeClientModel.VolumesList, error)
	GetAllVolumes(ctx context.Context, page, perPage int, filters ...string) (kubeClientModel.VolumesList, error)
	DeleteVolume(ctx context.Context, nsID, label string) error
	DeleteAllNamespaceVolumes(ctx context.Context, nsID string) error
	DeleteAllUserVolumes(ctx context.Context) error
}

var StandardVolumeFilter = database.VolumeFilter{
	NotDeleted: true,
}

const (
	ZeroUUID = "00000000-0000-0000-0000-000000000000"
)

func (s *Server) DirectCreateVolume(ctx context.Context, nsID string, req model.DirectVolumeCreateRequest) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"ns_id":    nsID,
		"capacity": req.Capacity,
		"label":    req.Label,
		"user_id":  userID,
	}).Infof("create volume")

	storage, getErr := s.db.StorageByName(ctx, req.Storage)
	if getErr != nil {
		return getErr
	}

	if storage.Size-storage.Used-req.Capacity < 0 {
		return errors.ErrNoFreeStorages()
	}

	volume := model.Volume{
		Resource: model.Resource{
			Label:       req.Label,
			OwnerUserID: userID,
		},
		Capacity:    req.Capacity,
		NamespaceID: nsID,
		StorageName: storage.Name,
	}

	err := s.db.Transactional(func(tx database.DB) error {
		if createErr := tx.CreateVolume(ctx, &volume); createErr != nil {
			return createErr
		}

		kubeVol := volume.ToKube()

		if createErr := s.clients.KubeAPI.CreateVolume(ctx, nsID, &kubeVol); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}

func (s *Server) ImportVolume(ctx context.Context, nsID string, req kubeClientModel.Volume) error {
	s.log.WithFields(logrus.Fields{
		"ns_id":    nsID,
		"capacity": req.Capacity,
		"label":    req.Name,
	}).Infof("import volume")

	err := s.db.Transactional(func(tx database.DB) error {
		storage, getErr := tx.StorageByName(ctx, req.StorageName)
		if getErr != nil {
			return getErr
		}

		if req.Owner == "" {
			req.Owner = ZeroUUID
		}

		volume := model.Volume{
			Resource: model.Resource{
				Label:       req.Name,
				OwnerUserID: req.Owner,
			},
			Capacity:    int(req.Capacity),
			NamespaceID: nsID,
			StorageName: storage.Name,
		}

		if createErr := tx.CreateVolume(ctx, &volume); createErr != nil {
			return createErr
		}

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

	freeVolume := req.TariffID == ZeroUUID

	var tariff billing.VolumeTariff

	if !freeVolume {
		var err error
		tariff, err = s.clients.Billing.GetVolumeTariff(ctx, req.TariffID)
		if err != nil {
			return err
		}

		if chkErr := CheckTariff(tariff.Tariff, IsAdminRole(ctx)); chkErr != nil {
			return chkErr
		}
	}

	storage, getErr := s.db.StorageByName(ctx, req.Storage)
	if getErr != nil {
		return getErr
	}

	var volume model.Volume
	if freeVolume {
		nsTariff, getErr := s.clients.Billing.GetTariffForNamespace(ctx, nsID)
		if getErr != nil {
			return getErr
		}

		if nsTariff.VolumeSize == 0 {
			return errors.ErrQuotaExceeded()
		}

		if storage.Size-storage.Used-nsTariff.VolumeSize < 0 {
			return errors.ErrNoFreeStorages()
		}

		volume = model.Volume{
			Resource: model.Resource{
				TariffID:    &req.TariffID,
				Label:       req.Label,
				OwnerUserID: userID,
			},
			Capacity:    nsTariff.VolumeSize,
			NamespaceID: nsID,
			StorageName: storage.Name,
		}
	} else {
		if storage.Size-storage.Used-tariff.StorageLimit < 0 {
			return errors.ErrNoFreeStorages()
		}

		volume = model.Volume{
			Resource: model.Resource{
				TariffID:    &req.TariffID,
				Label:       req.Label,
				OwnerUserID: userID,
				ID:          uuid.NewV4().String(),
			},
			Capacity:    tariff.StorageLimit,
			NamespaceID: nsID,
			StorageName: storage.Name,
		}

		subReq := billing.SubscribeTariffRequest{
			TariffID:      tariff.ID,
			ResourceType:  billing.Volume,
			ResourceLabel: volume.Label,
			ResourceID:    volume.ID,
		}
		if subErr := s.clients.Billing.Subscribe(ctx, subReq); subErr != nil {
			return subErr
		}
	}

	err := s.db.Transactional(func(tx database.DB) error {
		if createErr := tx.CreateVolume(ctx, &volume); createErr != nil {
			return createErr
		}

		kubeVol := volume.ToKube()

		if createErr := s.clients.KubeAPI.CreateVolume(ctx, nsID, &kubeVol); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}

func (s *Server) GetVolume(ctx context.Context, nsID, label string) (kubeClientModel.Volume, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
		"label":   label,
	}).Infof("get volume")

	vol, err := s.db.VolumeByLabel(ctx, nsID, label)
	if err != nil {
		return kubeClientModel.Volume{}, err
	}

	return vol.ToKube(), nil
}

func (s *Server) GetNamespaceVolumes(ctx context.Context, nsID string) (kubeClientModel.VolumesList, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"namespace_id": nsID,
	}).Infof("get namespace volumes")

	vols, err := s.db.NamespaceVolumes(ctx, nsID)
	if err != nil {
		return kubeClientModel.VolumesList{}, err
	}

	ret := make([]kubeClientModel.Volume, len(vols))
	for i := range vols {
		ret[i] = vols[i].ToKube()
	}

	return kubeClientModel.VolumesList{ret}, nil
}

func (s *Server) GetUserVolumes(ctx context.Context) (kubeClientModel.VolumesList, error) {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithField("user_id", userID).Infof("get user volumes")

	vols, err := s.db.UserVolumes(ctx, userID)
	if err != nil {
		return kubeClientModel.VolumesList{}, err
	}

	ret := make([]kubeClientModel.Volume, len(vols))
	for i := range vols {
		ret[i] = vols[i].ToKube()
	}

	return kubeClientModel.VolumesList{ret}, nil
}

func (s *Server) GetAllVolumes(ctx context.Context, page, perPage int, filters ...string) (kubeClientModel.VolumesList, error) {
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
		return kubeClientModel.VolumesList{}, err
	}

	ret := make([]kubeClientModel.Volume, len(vols))
	for i := range vols {
		ret[i] = vols[i].ToKube()
	}

	return kubeClientModel.VolumesList{ret}, nil
}

func (s *Server) DeleteVolume(ctx context.Context, nsID, label string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id": userID,
		"ns_id":   nsID,
		"label":   label,
	}).Infof("delete volume")

	err := s.db.Transactional(func(tx database.DB) error {
		vol, getErr := tx.VolumeByLabel(ctx, nsID, label)
		if getErr != nil {
			return getErr
		}

		if delErr := tx.DeleteVolume(ctx, &vol); delErr != nil {
			return delErr
		}

		if createErr := s.clients.KubeAPI.DeleteVolume(ctx, nsID, vol.Label); createErr != nil {
			return createErr
		}

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
		vols, err := s.db.UserVolumes(ctx, userID)
		switch {
		case err == nil:
			// pass
		case len(vols) == 0:
			return nil
		default:
			return err
		}

		if delErr := tx.DeleteVolumes(ctx, vols); delErr != nil {
			return delErr
		}

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

func (s *Server) DeleteAllNamespaceVolumes(ctx context.Context, nsID string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"namespace_id": nsID,
	}).Infof("delete all user volumes")

	err := s.db.Transactional(func(tx database.DB) error {
		vols, err := s.db.NamespaceVolumes(ctx, nsID)
		switch {
		case err == nil:
			// pass
		case len(vols) == 0:
			return nil
		default:
			return err
		}

		if delErr := tx.DeleteVolumes(ctx, vols); delErr != nil {
			return delErr
		}

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

func (s *Server) AdminResizeVolume(ctx context.Context, nsID, label string, newCapacity int) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":      userID,
		"ns_id":        nsID,
		"label":        label,
		"new_capacity": newCapacity,
	}).Infof("resize volume")

	err := s.db.Transactional(func(tx database.DB) error {
		vol, getErr := tx.VolumeByLabel(ctx, nsID, label)
		if getErr != nil {
			return getErr
		}

		if newCapacity < vol.Capacity {
			return errors.ErrDownResize()
		}

		vol.TariffID = nil
		vol.Capacity = newCapacity

		if resizeErr := tx.UpdateVolume(ctx, &vol); resizeErr != nil {
			return resizeErr
		}

		kubeVol := vol.ToKube()

		if createErr := s.clients.KubeAPI.UpdateVolume(ctx, nsID, &kubeVol); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}

func (s *Server) ResizeVolume(ctx context.Context, nsID, label string, newTariffID string) error {
	userID := httputil.MustGetUserID(ctx)
	s.log.WithFields(logrus.Fields{
		"user_id":       userID,
		"ns_id":         nsID,
		"label":         label,
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
		vol, getErr := tx.VolumeByLabel(ctx, nsID, label)
		if getErr != nil {
			return getErr
		}

		if newTariff.StorageLimit < vol.Capacity {
			return errors.ErrDownResize()
		}

		vol.TariffID = &newTariff.ID
		vol.Capacity = newTariff.StorageLimit

		if resizeErr := tx.UpdateVolume(ctx, &vol); resizeErr != nil {
			return resizeErr
		}

		kubeVol := vol.ToKube()

		if createErr := s.clients.KubeAPI.UpdateVolume(ctx, nsID, &kubeVol); createErr != nil {
			return createErr
		}

		return nil
	})

	return err
}
