package main

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"

	"git.containerum.net/ch/volume-manager/pkg/clients"
	"git.containerum.net/ch/volume-manager/pkg/database"
	"git.containerum.net/ch/volume-manager/pkg/database/postgres"
	"git.containerum.net/ch/volume-manager/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/en_US"
	"github.com/go-playground/universal-translator"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
)

type operationMode int

const (
	modeDebug operationMode = iota
	modeRelease
)

var opMode operationMode

func setupLogger(ctx *cli.Context) error {
	mode := ctx.String(ModeFlag.Name)
	switch mode {
	case "debug":
		opMode = modeDebug
		gin.SetMode(gin.DebugMode)
		logrus.SetLevel(logrus.DebugLevel)
	case "release", "":
		opMode = modeRelease
		gin.SetMode(gin.ReleaseMode)
		logrus.SetFormatter(&logrus.JSONFormatter{})

		level := logrus.Level(ctx.Int(LogLevelFlag.Name))
		if level > logrus.DebugLevel || level < logrus.PanicLevel {
			return errors.New("invalid log level")
		}
		logrus.SetLevel(level)
	default:
		return errors.New("invalid operation mode (must be 'debug' or 'release')")
	}
	return nil
}

func setupDB(ctx *cli.Context) (database.DB, error) {
	return postgres.Connect(fmt.Sprintf("postgres://%s%s@%s/%s?sslmode=%s",
		ctx.String(DBUserFlag.Name),
		func() string {
			if pass := ctx.String(DBPassFlag.Name); pass != "" {
				return ":" + pass
			}
			return ""
		}(),
		ctx.String(DBHostFlag.Name),
		ctx.String(DBBaseFlag.Name),
		func() string {
			if ctx.Bool(DBSSLModeFlag.Name) {
				return "enable"
			}
			return "disable"
		}()))
}

func getListenAddr(ctx *cli.Context) string {
	return ctx.String(ListenAddrFlag.Name)
}

func setupTranslator() *ut.UniversalTranslator {
	return ut.New(en.New(), en.New(), en_US.New())
}

func setupBillingClient(addr string) (clients.BillingClient, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewBillingDummyClient(), nil
	case addr != "":
		return clients.NewBillingHTTPClient(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for billing service")
	}
}

func setupKubeAPIClient(addr string) (clients.KubeAPIClient, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewKubeAPIDummyClient(), nil
	case addr != "":
		return clients.NewKubeAPIHTTPClient(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for billing service")
	}
}

func setupServiceClients(ctx *cli.Context) (*server.Clients, error) {
	var errs []error
	var serverClients server.Clients
	var err error

	if serverClients.Billing, err = setupBillingClient(ctx.String(BillingAddrFlag.Name)); err != nil {
		errs = append(errs, err)
	}
	if serverClients.KubeAPI, err = setupKubeAPIClient(ctx.String(KubeAPIAddrFlag.Name)); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("clients setup errors: %v", errs)
	}

	v := reflect.ValueOf(serverClients)
	for i := 0; i < reflect.TypeOf(serverClients).NumField(); i++ {
		f := v.Field(i)
		if str, ok := f.Interface().(fmt.Stringer); ok {
			logrus.Infof("%s", str)
		}
	}

	return &serverClients, nil
}
