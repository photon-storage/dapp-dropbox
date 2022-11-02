package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/photon-storage/go-common/log"
	pc "github.com/photon-storage/go-photon/config/config"

	"github.com/photo-storage/dropbox/api/server"
	"github.com/photo-storage/dropbox/api/service"
	"github.com/photo-storage/dropbox/config"
	"github.com/photo-storage/dropbox/database/mysql"
)

var (
	networkFlag = &cli.StringFlag{
		Name:     "network",
		Usage:    "Network type to boot the node as",
		Required: true,
	}

	//configPathFlag specifies the api config file path.
	configPathFlag = &cli.StringFlag{
		Name:     "config-file",
		Usage:    "The filepath to a json file, flag is required",
		Required: true,
	}

	// logLevelFlag defines the log level.
	logLevelFlag = &cli.StringFlag{
		Name:  "log-level",
		Usage: "Logging verbosity (trace, debug, info=default, warn, error, fatal, panic)",
		Value: "info",
	}

	// logFormatFlag specifies the log output format.
	logFormatFlag = &cli.StringFlag{
		Name:  "log-format",
		Usage: "Specify log formatting. Supports: text, json, fluentd, journald.",
		Value: "text",
	}

	// logFilenameFlag specifies the log output file name.
	logFilenameFlag = &cli.StringFlag{
		Name:  "log-file",
		Usage: "Specify log file name, relative or absolute",
	}

	// logColor specifies whether to force log color by skipping TTY check.
	logColorFlag = &cli.BoolFlag{
		Name:  "log-color",
		Usage: "Force log color to be enabled, skipping TTY check",
		Value: false,
	}
)

func main() {
	app := cli.App{
		Name:    "dropbox",
		Usage:   "this is a dropbox dapp for photon storage",
		Action:  action,
		Version: version(),
		Flags: []cli.Flag{
			networkFlag,
			configPathFlag,
			logLevelFlag,
			logFormatFlag,
			logFilenameFlag,
			logColorFlag,
		},
	}

	app.Before = func(ctx *cli.Context) error {
		logLvl, err := log.ParseLevel(ctx.String(logLevelFlag.Name))
		if err != nil {
			return err
		}

		logFmt, err := log.ParseFormat(ctx.String(logFormatFlag.Name))
		if err != nil {
			return err
		}

		if err := log.Init(logLvl, logFmt, false); err != nil {
			return err
		}

		logFilename := ctx.String(logFilenameFlag.Name)
		if logFilename != "" {
			if err := log.ConfigurePersistentLogging(logFilename, false); err != nil {
				log.Error("Failed to configuring logging to disk",
					"error", err)
			}
		}
		if ctx.Bool(logColorFlag.Name) {
			log.ForceColor()
		}

		configType, err := pc.ConfigTypeFromString(ctx.String(networkFlag.Name))
		if err != nil {
			return err
		}

		return pc.Use(configType)
	}

	if err := app.Run(os.Args); err != nil {
		log.Error("running api application failed", "error", err)
	}
}

func action(ctx *cli.Context) error {
	cfg := &Config{}
	if err := config.Load(ctx.String(configPathFlag.Name), cfg); err != nil {
		log.Fatal("reading api config failed", "error", err)
	}

	db, err := mysql.NewMySQLDB(cfg.MySQL)
	if err != nil {
		log.Fatal("initialize mysql db error", "error", err)
	}

	log.Info("Starting dropbox api server...")

	service, err := service.New(
		ctx.Context,
		db,
		cfg.NodeEndpoint,
		cfg.DepotBootstrap,
	)
	if err != nil {
		return err
	}

	server.New(cfg.Port, service).Run()
	return nil
}

// Config defines the config for api service.
type Config struct {
	Port           int          `yaml:"port"`
	MySQL          mysql.Config `yaml:"mysql"`
	NodeEndpoint   string       `yaml:"node_endpoint"`
	DepotBootstrap []string     `yaml:"depot_bootstrap"`
}
