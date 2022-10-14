package cmd

import (
	"context"
	"net/http"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/errors"
	pkgHttp "go.bryk.io/pkg/net/http"
	"go.bryk.io/pkg/otel"
)

var runCmd = &cobra.Command{
	Use:     "run",
	Short:   "Start a server instance to handle incoming requests",
	Example: "serve run -p 8080 ./html",
	RunE:    runServer,
}

func init() {
	if err := cli.SetupCommandParams(runCmd, conf.Overrides("server"), viper.GetViper()); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(runCmd)
}

func runServer(_ *cobra.Command, args []string) error {
	// validate provided content path
	var fp string
	if len(args) == 0 {
		var err error
		fp, err = os.Getwd()
		if err != nil {
			return errors.Errorf("can't use current directory: %s", err)
		}
	} else {
		fp = args[0]
	}

	// get observability operator
	oop, err := otel.NewOperator(conf.OTEL(log)...)
	if err != nil {
		return err
	}
	defer oop.Shutdown(context.TODO())

	// setup and start server
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(fp)))
	opts := conf.ServerOptions(mux, oop)
	srv, err := pkgHttp.NewServer(opts...)
	if err != nil {
		return err
	}
	go func() {
		_ = srv.Start()
	}()

	// wait for system signals
	log.Info("server is ready and waiting for requests")
	<-cli.SignalsHandler([]os.Signal{
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	})

	// close server
	log.Info("closing server")
	return srv.Stop(true)
}
