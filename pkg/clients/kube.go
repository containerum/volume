package clients

import (
	"context"
	"net/url"

	"git.containerum.net/ch/volume-manager/pkg/errors"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/containerum/utils/httputil"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type KubeAPIClient interface {
	CreateVolume(ctx context.Context, namespace string, volume *model.Volume) error
	UpdateVolume(ctx context.Context, namespace string, volume *model.Volume) error
	DeleteVolume(ctx context.Context, namespace string, volumeName string) error
}

type KubeAPIHTTPClient struct {
	log    *cherrylog.LogrusAdapter
	client *resty.Client
}

func NewKubeAPIHTTPClient(url *url.URL) *KubeAPIHTTPClient {
	log := logrus.WithField("component", "kube_api_client")

	client := resty.New().
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetHostURL(url.String()).
		SetDebug(true).
		SetError(cherry.Err{}).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return &KubeAPIHTTPClient{
		log:    cherrylog.NewLogrusAdapter(log),
		client: client,
	}
}

func (k *KubeAPIHTTPClient) CreateVolume(ctx context.Context, namespace string, volume *model.Volume) error {
	k.log.WithField("namespace", namespace).Debugf("create volume %+v", volume)

	resp, err := k.client.R().
		SetContext(ctx).
		SetBody(*volume).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetPathParams(map[string]string{
			"namespace": namespace,
		}).
		SetResult(volume).
		Post("/namespaces/{namespace}/volumes")
	if err != nil {
		return errors.ErrInternal().Log(err, k.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (k *KubeAPIHTTPClient) UpdateVolume(ctx context.Context, namespace string, volume *model.Volume) error {
	k.log.WithField("namespace", namespace).Debugf("update volume %+v")

	resp, err := k.client.R().
		SetContext(ctx).
		SetBody(*volume).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetPathParams(map[string]string{
			"namespace": namespace,
			"volume":    volume.Name,
		}).
		SetResult(volume).
		Put("/namespaces/{namespace}/volumes/{volume}")
	if err != nil {
		return errors.ErrInternal().Log(err, k.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (k *KubeAPIHTTPClient) DeleteVolume(ctx context.Context, namespace string, volumeName string) error {
	k.log.WithField("namespace", namespace).Debugf("delete volume %s", volumeName)

	resp, err := k.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetPathParams(map[string]string{
			"namespace": namespace,
			"volume":    volumeName,
		}).
		Delete("/namespaces/{namespace}/volumes/{volume}")
	if err != nil {
		return errors.ErrInternal().Log(err, k.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

type KubeAPIDummyClient struct {
	log *logrus.Entry
}

func NewKubeAPIDummyClient() *KubeAPIDummyClient {
	return &KubeAPIDummyClient{
		log: logrus.WithField("component", "kube_api_client"),
	}
}

func (k *KubeAPIDummyClient) CreateVolume(ctx context.Context, namespace string, volume *model.Volume) error {
	k.log.WithField("namespace", namespace).Debugf("create volume %+v", volume)

	return nil
}

func (k *KubeAPIDummyClient) UpdateVolume(ctx context.Context, namespace string, volume *model.Volume) error {
	k.log.WithField("namespace", namespace).Debugf("update volume %+v")

	return nil
}

func (k *KubeAPIDummyClient) DeleteVolume(ctx context.Context, namespace string, volumeName string) error {
	k.log.WithField("namespace", namespace).Debugf("delete volume %s", volumeName)

	return nil
}
