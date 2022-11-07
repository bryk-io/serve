package internal

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.bryk.io/pkg/cli"
	xlog "go.bryk.io/pkg/log"
	"go.bryk.io/pkg/net/csp"
	pkgHttp "go.bryk.io/pkg/net/http"
	"go.bryk.io/pkg/net/middleware"
	"go.bryk.io/pkg/otel"
	"go.bryk.io/pkg/otel/sentry"
)

// Default application/service identifier.
var appIdentifier = ""

// Settings provide utilities to manage configuration options available
// when utilizing the different components available through the CLI
// application.
type Settings struct {
	Server *serverSettings `json:"server" yaml:"server" mapstructure:"server"`
	Otel   *otelSettings   `json:"otel" yaml:"otel" mapstructure:"otel"`
}

// Load available config values from Viper into the settings instance.
func (s *Settings) Load(v *viper.Viper) error {
	return v.Unmarshal(s)
}

// SetDefaults loads default values to the provided viper instance.
func (s *Settings) SetDefaults(v *viper.Viper, appID string) {
	v.SetDefault("server.port", 9090)
	appIdentifier = appID
	if s.Otel == nil {
		s.Otel = &otelSettings{
			ServiceName: appIdentifier,
			Sentry:      &sentrySettings{},
		}
	}
	if s.Server == nil {
		s.Server = &serverSettings{
			Port:  9090,
			Cache: 3600,
			TLS:   &tlsSettings{Enabled: false},
			CSP:   &cspSettings{Enabled: false},
			Middleware: &mwSettings{
				Gzip:     5,
				Metadata: &mwMetadataSettings{Headers: []string{}},
				CORS:     new(middleware.CORSOptions),
			},
		}
	}
}

// Overrides return the available flag overrides for the command specified.
// Specific settings can be provided via: configuration file, ENV variable
// and command flags.
func (s *Settings) Overrides(cmd string) []cli.Param {
	switch cmd {
	case "server":
		return []cli.Param{
			{
				Name:      "port",
				Usage:     "HTTP port to use for the server",
				FlagKey:   "server.port",
				ByDefault: 9090,
				Short:     "p",
			},
			{
				Name:      "tls",
				Usage:     "enable secure communications using TLS with provided credentials",
				FlagKey:   "server.tls.enabled",
				ByDefault: false,
			},
			{
				Name:      "tls-ca",
				Usage:     "TLS custom certificate authority (path to PEM file)",
				FlagKey:   "server.tls.custom_ca",
				ByDefault: "",
			},
			{
				Name:      "tls-cert",
				Usage:     "TLS certificate (path to PEM file)",
				FlagKey:   "server.tls.cert",
				ByDefault: "/etc/serve/tls/tls.crt",
			},
			{
				Name:      "tls-key",
				Usage:     "server private key (path to PEM file)",
				FlagKey:   "server.tls.key",
				ByDefault: "/etc/serve/tls/tls.key",
			},
		}
	default:
		return []cli.Param{}
	}
}

// OTEL returns the configuration options available to set up an OTEL operator.
func (s *Settings) OTEL(log xlog.Logger) []otel.OperatorOption {
	opts := []otel.OperatorOption{
		otel.WithLogger(log),
		otel.WithServiceName(s.Otel.ServiceName),
		otel.WithServiceVersion(s.Otel.ServiceVersion),
		otel.WithResourceAttributes(s.Otel.Attributes),
		otel.WithHostMetrics(),
		otel.WithRuntimeMetrics(5 * time.Second),
	}
	if collector := s.Otel.Collector; collector != "" {
		opts = append(opts, otel.WithExporterOTLP(collector, true, nil)...)
	}

	// Error reporter
	if sentryInfo := s.Otel.Sentry; sentryInfo.DSN != "" {
		rep, err := sentry.Reporter(sentryInfo.DSN, sentryInfo.Env, s.Otel.ServiceVersion)
		if err == nil {
			opts = append(opts, otel.WithErrorReporter(rep))
		}
	}
	return opts
}

// ServerOptions returns the configuration options available to set up an
// HTTP server instance.
func (s *Settings) ServerOptions(handler http.Handler, log xlog.Logger) []pkgHttp.Option {
	opts := []pkgHttp.Option{
		pkgHttp.WithHandler(handler),
		pkgHttp.WithPort(s.Server.Port),
		pkgHttp.WithIdleTimeout(10 * time.Second),
		pkgHttp.WithMiddleware(middleware.CORS(*s.Server.Middleware.CORS)),
		pkgHttp.WithMiddleware(middleware.ContextMetadata(s.MetadataOptions())),
		pkgHttp.WithMiddleware(middleware.GzipCompression(s.Server.Middleware.Gzip)),
		pkgHttp.WithMiddleware(middleware.Logging(log, nil)),
		pkgHttp.WithMiddleware(s.extraHeaders),
		pkgHttp.WithMiddleware(middleware.PanicRecovery()),
	}
	if s.Server.ProxyProtocol {
		opts = append(opts, pkgHttp.WithMiddleware(middleware.ProxyHeaders()))
	}
	if s.Server.CSP.Enabled {
		var cspOpts []csp.Option
		if s.Server.CSP.AllowEval {
			cspOpts = append(cspOpts, csp.UnsafeEval())
		}
		if s.Server.CSP.ReportOnly {
			cspOpts = append(cspOpts, csp.WithReportOnly())
		}
		if len(s.Server.CSP.ReportTo) > 0 {
			cspOpts = append(cspOpts, csp.WithReportTo(s.Server.CSP.ReportTo...))
		}
		policy, _ := csp.New(cspOpts...)
		opts = append(opts, pkgHttp.WithMiddleware(policy.Handler()))
	}
	if s.Server.TLS.Enabled {
		if err := expandTLS(s.Server.TLS); err == nil {
			opts = append(opts, pkgHttp.WithTLS(pkgHttp.TLS{
				Cert:             s.Server.TLS.cert,
				PrivateKey:       s.Server.TLS.key,
				IncludeSystemCAs: s.Server.TLS.SystemCA,
				CustomCAs:        s.Server.TLS.customCAs,
			}))
		}
	}
	return opts
}

// ReleaseCode returns the release identifier for the application. A release
// identifier is of the form: `service-name@version+commit_hash`. If `version`
// or `commit_hash` are not available will be omitted.
func (s *Settings) ReleaseCode() string {
	// use service name. prefer a manually provided value and fallback to the
	// hardcoded application identifier.
	release := appIdentifier
	if s.Otel.ServiceName != "" {
		release = s.Otel.ServiceName
	}

	// attach version tag. manually set value by default but prefer the one set
	// at build time if available
	version := s.Otel.ServiceVersion
	if strings.Count(CoreVersion, ".") >= 2 {
		version = CoreVersion
	}
	if version != "" {
		release = fmt.Sprintf("%s@%s", release, version)
	}

	// attach commit hash if available
	if BuildCode != "" {
		release = fmt.Sprintf("%s+%s", release, BuildCode)
	}
	return release
}

// CORS provides a "Cross Origin Resource Sharing" middleware.
func (s *Settings) CORS() middleware.CORSOptions {
	return *s.Server.Middleware.CORS
}

// MetadataOptions return configuration settings required to adjust the behavior
// of the metadata middleware.
func (s *Settings) MetadataOptions() middleware.ContextMetadataOptions {
	return middleware.ContextMetadataOptions{Headers: s.Server.Middleware.Metadata.Headers}
}

// Custom middleware to add additional headers.
func (s *Settings) extraHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Server.Cache > 0 {
			w.Header().Add("Cache-Control", fmt.Sprintf("public, max-age=%d", s.Server.Cache))
		}
		w.Header().Set("x-serve-version", CoreVersion)
		w.Header().Set("x-serve-build", BuildCode)
		w.Header().Set("x-serve-release", s.ReleaseCode())
		next.ServeHTTP(w, r)
	})
}

type serverSettings struct {
	Port          int          `json:"port" yaml:"port" mapstructure:"port"`
	Cache         uint         `json:"cache" yaml:"cache" mapstructure:"cache"`
	ProxyProtocol bool         `json:"proxy_protocol" yaml:"proxy_protocol" mapstructure:"proxy_protocol"`
	TLS           *tlsSettings `json:"tls" yaml:"tls" mapstructure:"tls"`
	CSP           *cspSettings `json:"csp" yaml:"csp" mapstructure:"csp"`
	Middleware    *mwSettings  `json:"middleware" yaml:"middleware" mapstructure:"middleware"`
}

type otelSettings struct {
	ServiceName    string                 `json:"service_name" yaml:"service_name" mapstructure:"service_name"`
	ServiceVersion string                 `json:"service_version" yaml:"service_version" mapstructure:"service_version"`
	Collector      string                 `json:"collector" yaml:"collector" mapstructure:"collector"`
	Attributes     map[string]interface{} `json:"attributes" yaml:"attributes" mapstructure:"attributes"`
	Sentry         *sentrySettings        `json:"sentry" yaml:"sentry" mapstructure:"sentry"`
}

type sentrySettings struct {
	DSN string `json:"dsn" yaml:"dsn" mapstructure:"dsn"`
	Env string `json:"environment" yaml:"environment" mapstructure:"environment"`
}

type tlsSettings struct {
	Enabled  bool     `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	SystemCA bool     `json:"system_ca" yaml:"system_ca" mapstructure:"system_ca"`
	Cert     string   `json:"cert" yaml:"cert" mapstructure:"cert"`
	Key      string   `json:"key" yaml:"key" mapstructure:"key"`
	CustomCA []string `json:"custom_ca" yaml:"custom_ca" mapstructure:"custom_ca"`

	// private expanded values
	cert      []byte
	key       []byte
	customCAs [][]byte
}

type cspSettings struct {
	Enabled    bool     `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	AllowEval  bool     `json:"allow_eval" yaml:"allow_eval" mapstructure:"allow_eval"`
	ReportOnly bool     `json:"report_only" yaml:"report_only" mapstructure:"report_only"`
	ReportTo   []string `json:"report_to" yaml:"report_to" mapstructure:"report_to"`
}

type mwSettings struct {
	Gzip     int                     `json:"gzip" yaml:"gzip" mapstructure:"gzip"`
	Metadata *mwMetadataSettings     `json:"metadata" yaml:"metadata" mapstructure:"metadata"`
	CORS     *middleware.CORSOptions `json:"cors" yaml:"cors" mapstructure:"cors"`
}

type mwMetadataSettings struct {
	Headers []string `json:"headers" yaml:"headers" mapstructure:"headers"`
}
