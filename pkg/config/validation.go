package config

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/nite-coder/bifrost/pkg/dns"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
)

// ValidateConfig checks if the config's values are valid, but does not check if the config's value mapping is valid
func ValidateConfig(mainOpts Options) error {

	err := validateTracing(mainOpts.Tracing)
	if err != nil {
		return err
	}

	err = validateLogging(mainOpts.Logging)
	if err != nil {
		return err
	}

	err = validateAccessLog(mainOpts.AccessLogs)
	if err != nil {
		return err
	}

	err = validateServers(mainOpts)
	if err != nil {
		return err
	}

	err = validateRoutes(mainOpts.Routes)
	if err != nil {
		return err
	}

	err = validateUpstreams(mainOpts.Upstreams)
	if err != nil {
		return err
	}

	if len(mainOpts.Servers) == 0 {
		return errors.New("no server found.  please add one server at lease")
	}

	for serverID, server := range mainOpts.Servers {
		for _, m := range server.Middlewares {
			if len(m.Use) > 0 {
				if _, found := mainOpts.Middlewares[m.Use]; !found {
					return fmt.Errorf("the middleware '%s' can't be found in the server '%s'", m.Use, serverID)
				}
			}

			if len(m.Type) > 0 {
				hander := middleware.FindHandlerByType(m.Type)
				if hander == nil {
					return fmt.Errorf("the middleware '%s' can't be found in the server '%s'", m.Type, serverID)
				}
			}
		}
	}

	for routeID, route := range mainOpts.Routes {
		for _, serverID := range route.Servers {
			if _, found := mainOpts.Servers[serverID]; !found {
				return fmt.Errorf("the server '%s' can't be found in the route '%s'", serverID, routeID)
			}
		}

		for _, middleware := range route.Middlewares {
			if len(middleware.Use) > 0 {
				if _, found := mainOpts.Middlewares[middleware.Use]; !found {
					return fmt.Errorf("the middleware '%s' can't be found in the route '%s'", middleware.Use, routeID)
				}
			}
		}
	}

	for serviceID, service := range mainOpts.Services {
		for _, m := range service.Middlewares {
			if len(m.Use) > 0 {
				if _, found := mainOpts.Middlewares[m.Use]; !found {
					return fmt.Errorf("the middleware '%s' can't be found in the service '%s'", m.Use, serviceID)
				}
			}

			if len(m.Type) > 0 {
				hander := middleware.FindHandlerByType(m.Type)
				if hander == nil {
					return fmt.Errorf("the middleware '%s' can't be found in the service '%s'", m.Type, serviceID)
				}
			}
		}
	}

	err = validateMetrics(mainOpts)
	if err != nil {
		return err
	}

	err = validateFQDN(mainOpts)
	if err != nil {
		return err
	}

	return nil
}

func validateLogging(opts LoggingOtions) error {

	level := strings.ToLower(opts.Level)
	switch level {
	case "", "debug", "info", "warn", "error":
	default:
		msg := fmt.Sprintf("logging level '%s' is not supported", level)
		fullpath := []string{"logging", "level"}
		return newInvalidConfig(fullpath, level, msg)
	}

	handler := strings.ToLower(opts.Handler)
	switch handler {
	case "text", "json", "":
	default:
		msg := fmt.Sprintf("logging handler '%s' is not supported", opts.Handler)
		fullpath := []string{"logging", "handler"}
		return newInvalidConfig(fullpath, opts.Handler, msg)
	}

	return nil
}

func validateAccessLog(options map[string]AccessLogOptions) error {
	reIsVariable := regexp.MustCompile(`\$\w+(?:[._-]\w+)*`)

	for id, opt := range options {
		if opt.Template == "" {
			msg := fmt.Sprintf("the template can't be empty for access log '%s'", id)
			fullpath := []string{"access_logs", id, "template"}
			return newInvalidConfig(fullpath, opt.Template, msg)
		}

		if len(opt.TimeFormat) > 0 {
			_, err := time.Parse(opt.TimeFormat, time.Now().Format(opt.TimeFormat))
			if err != nil {
				msg := fmt.Sprintf("the time format '%s' for access log '%s' is invalid", opt.TimeFormat, id)
				fullpath := []string{"access_logs", id, "time_format"}
				return newInvalidConfig(fullpath, opt.TimeFormat, msg)
			}
		}

		if len(opt.Escape) > 0 {
			switch opt.Escape {
			case "json", "none", "default", "":
			default:
				msg := fmt.Sprintf("the escape '%s' for access log '%s' is invalid", opt.Escape, id)
				fullpath := []string{"access_logs", id, "escape"}
				return newInvalidConfig(fullpath, opt.Escape, msg)
			}
		}

		directives := reIsVariable.FindAllString(opt.Template, -1)

		for _, directive := range directives {
			if !variable.IsDirective(directive) {
				return fmt.Errorf("the directive '%s' is not supported in the access log '%s'", directive, id)
			}
		}
	}

	return nil
}

func validateServers(mainOptions Options) error {

	for id, serverOptions := range mainOptions.Servers {
		if serverOptions.Bind == "" {
			msg := fmt.Sprintf("the bind can't be empty for server '%s'", id)
			fullpath := []string{"servers", id, "bind"}
			return newInvalidConfig(fullpath, "", msg)
		}

		if len(serverOptions.TLS.CertPEM) > 0 {
			_, err := os.ReadFile(serverOptions.TLS.CertPEM)
			if err != nil {
				msg := fmt.Sprintf("the cert pem file is invalid for server '%s'", id)

				if os.IsNotExist(err) {
					msg = fmt.Sprintf("the cert pem file doesn't exist for server '%s'", id)
				}

				fullpath := []string{"servers", id, "tls", "cert_pem"}
				return newInvalidConfig(fullpath, serverOptions.TLS.CertPEM, msg)
			}
		}

		if len(serverOptions.TLS.KeyPEM) > 0 {
			_, err := os.ReadFile(serverOptions.TLS.KeyPEM)
			if err != nil {
				msg := fmt.Sprintf("the key pem file is invalid for server '%s'", id)

				if os.IsNotExist(err) {
					msg = fmt.Sprintf("the key pem file doesn't exist for server '%s'", id)
				}

				fullpath := []string{"servers", id, "tls", "key_pem"}
				return newInvalidConfig(fullpath, serverOptions.TLS.KeyPEM, msg)
			}
		}

		if serverOptions.AccessLogID != "" {
			if _, found := mainOptions.AccessLogs[serverOptions.AccessLogID]; !found {
				msg := fmt.Sprintf("the access log '%s' doesn't exist for server '%s'", serverOptions.AccessLogID, id)
				fullpath := []string{"servers", id, "access_log_id"}
				return newInvalidConfig(fullpath, serverOptions.AccessLogID, msg)
			}
		}
	}

	return nil
}

func validateRoutes(options map[string]RouteOptions) error {
	for id, opt := range options {
		if opt.ServiceID == "" {
			msg := fmt.Sprintf("the 'service_id' can't be empty for the route '%s'", id)
			fullpath := []string{"routes", id, "service_id"}
			return newInvalidConfig(fullpath, "", msg)
		}
	}

	return nil
}

func validateUpstreams(options map[string]UpstreamOptions) error {
	for upstreamID, opt := range options {

		if upstreamID[0] == '$' {
			msg := fmt.Sprintf("the upstream '%s' can't start with '$'", upstreamID)
			fullpath := []string{"upstreams", upstreamID}
			return newInvalidConfig(fullpath, "", msg)
		}

		switch opt.Strategy {
		case WeightedStrategy, RandomStrategy, HashingStrategy, RoundRobinStrategy, "":
		default:
			msg := fmt.Sprintf("the strategy '%s' for the upstream '%s' is not supported", opt.Strategy, upstreamID)
			fullpath := []string{"upstreams", upstreamID, "strategy"}
			return newInvalidConfig(fullpath, opt.Strategy, msg)
		}

		if opt.Strategy == HashingStrategy && opt.HashOn == "" {
			msg := fmt.Sprintf("the hash_on can't be empty for the upstream '%s'", upstreamID)
			fullpath := []string{"upstreams", upstreamID, "hash_on"}
			return newInvalidConfig(fullpath, "", msg)
		}
	}

	return nil
}

func validateMetrics(options Options) error {
	if options.Metrics.Prometheus.Enabled {
		if options.Metrics.Prometheus.ServerID == "" {
			return errors.New("the server_id can't be empty for the prometheus")
		}

		_, found := options.Servers[options.Metrics.Prometheus.ServerID]
		if !found {
			return fmt.Errorf("the server_id '%s' for the prometheus is not found", options.Metrics.Prometheus.ServerID)
		}
	}

	return nil
}

func validateFQDN(mainOpts Options) error {
	if mainOpts.Resolver.SkipTest {
		return nil
	}

	// check target FQDN
	resolver, err := dns.NewResolver(dns.Options{
		AddrPort: mainOpts.Resolver.AddrPort,
		Valid:    mainOpts.Resolver.Valid,
	})

	if err != nil {
		return err
	}

	for _, upstream := range mainOpts.Upstreams {
		for _, target := range upstream.Targets {
			addr := extractAddr(target.Target)
			_, err := resolver.Lookup(context.Background(), addr)
			if err != nil {
				return err
			}
		}
	}

	for serviceID, service := range mainOpts.Services {
		addr, err := url.Parse(service.Url)
		if err != nil {
			return err
		}

		hostname := addr.Hostname()

		// validate
		if len(hostname) == 0 {
			return fmt.Errorf("the hostname is invalid in service url. service_id: %s", serviceID)
		}

		// dynamic upstream
		if hostname[0] == '$' {
			continue
		}

		// exist upstream
		_, found := mainOpts.Upstreams[hostname]
		if found {
			continue
		}

		host := extractAddr(hostname)
		_, err = resolver.Lookup(context.Background(), host)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractAddr(addrport string) string {
	parts := strings.Split(addrport, ":")

	if len(parts) == 1 {
		return addrport
	}

	return parts[0]
}

func validateTracing(opts TracingOptions) error {

	if !opts.Enabled {
		return nil
	}

	if opts.ServiceName == "" {
		return errors.New("the service_name can't be empty for the tracing")
	}

	for _, propagator := range opts.Propagators {
		switch propagator {
		case "", "b3", "tracecontext", "baggage", "jaeger":
		default:
			return fmt.Errorf("the propagator '%s' is not supported in tracing", propagator)
		}
	}

	return nil
}
