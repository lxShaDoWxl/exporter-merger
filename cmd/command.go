package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var config = &Config{}

func NewRootCommand() *cobra.Command {
	app := new(App)

	cmd := &cobra.Command{
		Use:   "exporter-merger",
		Short: "merges Prometheus metrics from multiple sources",
		Run:   app.run,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if app.viper.GetBool("verbose") {
				log.SetLevel(log.DebugLevel)
			} else {
				log.SetLevel(log.InfoLevel)
			}
		},
	}

	app.Bind(cmd)

	cmd.AddCommand(NewVersionCommand())

	return cmd
}

type App struct {
	viper *viper.Viper
}

func (app *App) Bind(cmd *cobra.Command) {
	app.viper = viper.New()
	app.viper.SetEnvPrefix("MERGER")
	app.viper.AutomaticEnv()

	configPath := cmd.PersistentFlags().StringP(
		"config-path", "c", "/etc/exporter-merger/config.yaml",
		"Path to the configuration file.")
	cobra.OnInitialize(func() {
		var err error
		if configPath != nil && *configPath != "" {
			config, err = ReadConfig(*configPath)
			if err != nil {
				log.WithField("error", err).Errorf("failed to load config file '%s'", *configPath)
				os.Exit(1)
				return
			}
		}
	})

	cmd.PersistentFlags().Int(
		"listen-port", 8080,
		"Listen port for the HTTP server. (ENV:MERGER_PORT)")
	err := app.viper.BindPFlag("port", cmd.PersistentFlags().Lookup("listen-port"))
	if err != nil {
		log.WithField("error", err).Errorf("failed flag 'port'")
		os.Exit(1)
		return
	}

	cmd.PersistentFlags().Int(
		"exporters-timeout", 10,
		"HTTP client timeout for connecting to exporters. (ENV:MERGER_EXPORTERSTIMEOUT)")
	err = app.viper.BindPFlag("exporterstimeout", cmd.PersistentFlags().Lookup("exporters-timeout"))
	if err != nil {
		log.WithField("error", err).Errorf("failed flag 'exporterstimeout'")
		os.Exit(1)
		return
	}

	cmd.PersistentFlags().BoolP(
		"verbose", "v", false,
		"Include debug messages to output (ENV:MERGER_VERBOSE)")
	err = app.viper.BindPFlag("verbose", cmd.PersistentFlags().Lookup("verbose"))
	if err != nil {
		log.WithField("error", err).Errorf("failed flag 'verbose'")
		os.Exit(1)
		return
	}
}

func (app *App) run(_ *cobra.Command, _ []string) {
	http.Handle("/metrics", Handler{
		Exporters:            config.Exporters,
		ExportersHTTPTimeout: app.viper.GetInt("exporterstimeout"),
	})

	port := app.viper.GetInt("port")
	log.Infof("starting HTTP server on port %d", port)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		ReadHeaderTimeout: 3 * time.Second,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
