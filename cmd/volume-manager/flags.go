package main

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
)

var (
	ModeFlag = cli.StringFlag{
		Name:    "mode",
		EnvVars: []string{"MODE"},
		Value:   "debug",
	}

	LogLevelFlag = cli.IntFlag{
		Name:    "log_level",
		EnvVars: []string{"LOG_LEVEL"},
		Value:   int(logrus.InfoLevel),
	}

	DBUserFlag = cli.StringFlag{
		Name:    "db_user",
		EnvVars: []string{"DB_USER"},
		Value:   "postgres",
	}

	DBPassFlag = cli.StringFlag{
		Name:    "db_pass",
		EnvVars: []string{"DB_PASSWORD"},
		Value:   "postgres",
	}

	DBHostFlag = cli.StringFlag{
		Name:    "db_host",
		EnvVars: []string{"DB_HOST"},
		Value:   "localhost:5432",
	}

	DBBaseFlag = cli.StringFlag{
		Name:    "db_base",
		EnvVars: []string{"DB_BASE"},
		Value:   "permissions",
	}

	DBSSLModeFlag = cli.BoolFlag{
		Name:    "db_sslmode",
		EnvVars: []string{"DB_SSLMODE"},
	}

	ListenAddrFlag = cli.StringFlag{
		Name:    "listen_addr",
		EnvVars: []string{"LISTEN_ADDR"},
		Value:   ":4242",
	}

	BillingAddrFlag = cli.StringFlag{
		Name:    "billing_addr",
		EnvVars: []string{"BILLING_ADDR"},
	}

	CORSFlag = cli.BoolFlag{
		Name: "cors",
	}
)
