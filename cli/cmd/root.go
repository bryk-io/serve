package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bryk-io/serve/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.bryk.io/pkg/errors"
	xlog "go.bryk.io/pkg/log"
)

var (
	log     xlog.Logger        // app main logger
	conf    *internal.Settings // app settings management
	cfgFile = ""               // configuration file used
)

var rootCmd = &cobra.Command{
	Use:           "serve",
	Short:         "Basic file server with batteries included",
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: strings.TrimSpace(`
Basic file server with batteries included.

Deploy a website from a local filesystem with support for
logging, OpenTelemetry, CORS, HSTS, cache and more.`),
}

// Execute provides the main entry point for the application.
func Execute() {
	// catch any panics
	defer func() {
		if err := errors.FromRecover(recover()); err != nil {
			log.Warning("recovered panic")
			fmt.Printf("%+v", err)
			os.Exit(1)
		}
	}()
	// execute command
	if err := rootCmd.Execute(); err != nil {
		if pe := new(errors.Error); errors.Is(err, pe) {
			log.WithField("error", err).Error("command failed")
		} else {
			log.Error(err)
		}
		os.Exit(1)
	}
}

func init() {
	log = xlog.WithZero(xlog.ZeroOptions{PrettyPrint: true})
	conf = new(internal.Settings)
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}

func initConfig() {
	// Used for ENV variables prefix and home directories
	var appIdentifier = "serve"

	// Load configuration defaults
	conf.SetDefaults(viper.GetViper(), appIdentifier)

	// Set configuration file
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(filepath.Join("/etc", appIdentifier))
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(filepath.Join(home, appIdentifier))
			viper.AddConfigPath(filepath.Join(home, fmt.Sprintf(".%s", appIdentifier)))
		}
		viper.AddConfigPath(".")
	}

	// ENV
	viper.SetEnvPrefix(appIdentifier)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil && viper.ConfigFileUsed() != "" {
		log.WithField("file", viper.ConfigFileUsed()).Error("failed to load configuration file")
	}
	if cf := viper.ConfigFileUsed(); cf != "" {
		log.WithField("file", cf).Info("configuration loaded")
	}

	// Load configuration into "settings" helper
	if err := conf.Load(viper.GetViper()); err != nil {
		panic(err)
	}
}
