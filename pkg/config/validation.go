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
func ValidateConfig(mainOpts Options, isFullMode bool) error {

	err := validateLogging(mainOpts.Logging)
	if err != nil {
		return err
	}

	err = validateTracing(mainOpts.Tracing)
	if err != nil {
		return err
	}

	err = validateAccessLog(mainOpts.AccessLogs)
	if err != nil {
		return err
	}

	err = validateUpstreams(mainOpts.Upstreams)
	if err != nil {
		return err
	}

	err = validateServices(mainOpts, isFullMode)
	if err != nil {
		return err
	}

	err = validateRoutes(mainOpts, isFullMode)
	if err != nil {
		return err
	}

	err = validateServers(mainOpts, isFullMode)
	if err != nil {
		return err
	}

	err = validateMetrics(mainOpts, isFullMode)
	if err != nil {
		return err
	}

	err = validateFQDN(mainOpts, isFullMode)
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
		structure := []string{"logging", "level"}
		return newInvalidConfig(structure, level, msg)
	}

	handler := strings.ToLower(opts.Handler)
	switch handler {
	case "text", "json", "":
	default:
		msg := fmt.Sprintf("logging handler '%s' is not supported", opts.Handler)
		structure := []string{"logging", "handler"}
		return newInvalidConfig(structure, opts.Handler, msg)
	}

	return nil
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
		case "b3", "tracecontext", "baggage", "jaeger": // ok
		case "":
			return errors.New("the propagator can't be empty for the tracing")
		default:
			return fmt.Errorf("the propagator '%s' is not supported in tracing", propagator)
		}
	}

	return nil
}

func validateAccessLog(options map[string]AccessLogOptions) error {
	reIsVariable := regexp.MustCompile(`\$\w+(?:[._-]\w+)*`)

	for id, opt := range options {
		if opt.Template == "" {
			msg := fmt.Sprintf("the template can't be empty for access log '%s'", id)
			structure := []string{"access_logs", id, "template"}
			return newInvalidConfig(structure, opt.Template, msg)
		}

		if len(opt.TimeFormat) > 0 {
			_, err := time.Parse(opt.TimeFormat, time.Now().Format(opt.TimeFormat))
			if err != nil {
				msg := fmt.Sprintf("the time format '%s' for access log '%s' is invalid", opt.TimeFormat, id)
				structure := []string{"access_logs", id, "time_format"}
				return newInvalidConfig(structure, opt.TimeFormat, msg)
			}
		}

		if len(opt.Escape) > 0 {
			switch opt.Escape {
			case "json", "none", "default", "":
			default:
				msg := fmt.Sprintf("the escape '%s' for access log '%s' is invalid", opt.Escape, id)
				structure := []string{"access_logs", id, "escape"}
				return newInvalidConfig(structure, opt.Escape, msg)
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

func validateServers(mainOptions Options, isFullMode bool) error {

	for serverID, serverOptions := range mainOptions.Servers {
		if serverOptions.Bind == "" {
			msg := fmt.Sprintf("the bind can't be empty for server '%s'", serverID)
			structure := []string{"servers", serverID, "bind"}
			return newInvalidConfig(structure, "", msg)
		}

		if len(serverOptions.TLS.CertPEM) > 0 {
			_, err := os.ReadFile(serverOptions.TLS.CertPEM)
			if err != nil {
				msg := fmt.Sprintf("the cert pem file is invalid for server '%s'", serverID)

				if os.IsNotExist(err) {
					msg = fmt.Sprintf("the cert pem file doesn't exist for server '%s'", serverID)
				}

				structure := []string{"servers", serverID, "tls", "cert_pem"}
				return newInvalidConfig(structure, serverOptions.TLS.CertPEM, msg)
			}
		}

		if len(serverOptions.TLS.KeyPEM) > 0 {
			_, err := os.ReadFile(serverOptions.TLS.KeyPEM)
			if err != nil {
				msg := fmt.Sprintf("the key pem file is invalid for server '%s'", serverID)

				if os.IsNotExist(err) {
					msg = fmt.Sprintf("the key pem file doesn't exist for server '%s'", serverID)
				}

				structure := []string{"servers", serverID, "tls", "key_pem"}
				return newInvalidConfig(structure, serverOptions.TLS.KeyPEM, msg)
			}
		}

		if serverOptions.AccessLogID != "" {
			if _, found := mainOptions.AccessLogs[serverOptions.AccessLogID]; !found {
				msg := fmt.Sprintf("the access log '%s' doesn't exist for server '%s'", serverOptions.AccessLogID, serverID)
				structure := []string{"servers", serverID, "access_log_id"}
				return newInvalidConfig(structure, serverOptions.AccessLogID, msg)
			}
		}

		if isFullMode {
			for _, m := range serverOptions.Middlewares {
				if len(m.Use) > 0 {
					if _, found := mainOptions.Middlewares[m.Use]; !found {
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
	}

	return nil
}

func validateRoutes(mainOptions Options, isFullMode bool) error {

	for routeID, route := range mainOptions.Routes {
		if route.ServiceID == "" {
			msg := fmt.Sprintf("the 'service_id' can't be empty in the route '%s'", routeID)
			structure := []string{"routes", routeID, "service_id"}
			return newInvalidConfig(structure, "", msg)
		}

		if len(route.Paths) == 0 {
			msg := fmt.Sprintf("the paths can't be empty in the route '%s'", routeID)
			structure := []string{"routes", routeID, "paths"}
			return newInvalidConfig(structure, "", msg)
		}

		if !isFullMode {
			continue
		}

		if _, found := mainOptions.Services[route.ServiceID]; !found {
			msg := fmt.Sprintf("the service '%s' can't be found in the route '%s'", route.ServiceID, routeID)
			structure := []string{"routes", routeID, "service_id"}
			return newInvalidConfig(structure, "", msg)
		}

		for _, serverID := range route.Servers {
			if _, found := mainOptions.Servers[serverID]; !found {
				msg := fmt.Sprintf("the server '%s' can't be found in the route '%s'", serverID, routeID)
				structure := []string{"routes", routeID, "servers"}
				return newInvalidConfig(structure, serverID, msg)
			}
		}

		for _, middleware := range route.Middlewares {
			if len(middleware.Use) > 0 {
				if _, found := mainOptions.Middlewares[middleware.Use]; !found {
					msg := fmt.Sprintf("the middleware '%s' can't be found in the route '%s'", middleware.Use, routeID)
					structure := []string{"routes", routeID, "middlewares"}
					return newInvalidConfig(structure, middleware.Use, msg)
				}
			}
		}
	}

	return nil
}

func validateServices(mainOptions Options, isFullMode bool) error {

	for serviceID, service := range mainOptions.Services {

		if !isFullMode {
			continue
		}

		addr, err := url.Parse(service.Url)
		if err != nil {
			return err
		}

		hostname := addr.Hostname()

		// validate
		if len(hostname) == 0 {
			return fmt.Errorf("the host is invalid in service url. service_id: %s", serviceID)
		}

		// exist upstream
		if hostname[0] != '$' {
			_, found := mainOptions.Upstreams[hostname]
			if !found && (!IsValidDomain(hostname)) {
				return fmt.Errorf("the upstream '%s' can't be found in the service '%s'", hostname, serviceID)
			}
		}

		for _, m := range service.Middlewares {
			if len(m.Use) > 0 {
				if _, found := mainOptions.Middlewares[m.Use]; !found {
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

	return nil
}

func validateUpstreams(options map[string]UpstreamOptions) error {
	for upstreamID, opt := range options {

		if upstreamID[0] == '$' {
			msg := fmt.Sprintf("the upstream '%s' can't start with '$'", upstreamID)
			structure := []string{"upstreams", upstreamID}
			return newInvalidConfig(structure, "", msg)
		}

		switch opt.Strategy {
		case WeightedStrategy, RandomStrategy, HashingStrategy, RoundRobinStrategy, "":
		default:
			msg := fmt.Sprintf("the strategy '%s' for the upstream '%s' is not supported", opt.Strategy, upstreamID)
			structure := []string{"upstreams", upstreamID, "strategy"}
			return newInvalidConfig(structure, opt.Strategy, msg)
		}

		if opt.Strategy == HashingStrategy && opt.HashOn == "" {
			msg := fmt.Sprintf("the hash_on can't be empty in the upstream '%s'", upstreamID)
			structure := []string{"upstreams", upstreamID, "hash_on"}
			return newInvalidConfig(structure, "", msg)
		}

		if len(opt.Targets) == 0 {
			return fmt.Errorf("the targets can't be empty in the upstream '%s'", upstreamID)
		}
	}

	return nil
}

func validateMetrics(options Options, isFullMode bool) error {
	if options.Metrics.Prometheus.Enabled {
		if options.Metrics.Prometheus.ServerID == "" {
			return errors.New("the server_id can't be empty for the prometheus")
		}

		if !isFullMode {
			return nil
		}

		_, found := options.Servers[options.Metrics.Prometheus.ServerID]
		if !found {
			msg := fmt.Sprintf("the server_id '%s' for the prometheus is not found", options.Metrics.Prometheus.ServerID)
			structure := []string{"metrics", "prometheus", "server_id"}
			return newInvalidConfig(structure, options.Metrics.Prometheus.ServerID, msg)
		}
	}

	return nil
}

func validateFQDN(mainOpts Options, isFullMode bool) error {
	if mainOpts.Resolver.SkipTest || !isFullMode {
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
