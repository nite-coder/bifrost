package config

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/dns"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/router"
	"github.com/nite-coder/bifrost/pkg/variable"
)

// ValidateConfig checks if the config's values are valid, but does not check if the config's value mapping is valid
func ValidateConfig(mainOptions Options, isFullMode bool) error {

	if dnsResolver == nil && !mainOptions.Resolver.SkipTest {
		var err error
		dnsResolver, err = dns.NewResolver(dns.Options{
			AddrPort: mainOptions.Resolver.AddrPort,
		})

		if err != nil {
			return err
		}
	}

	err := validateLogging(mainOptions.Logging)
	if err != nil {
		return err
	}

	err = validateTracing(mainOptions.Tracing)
	if err != nil {
		return err
	}

	err = validateProviders(mainOptions)
	if err != nil {
		return err
	}

	err = validateAccessLog(mainOptions.AccessLogs)
	if err != nil {
		return err
	}

	err = validateUpstreams(mainOptions, isFullMode)
	if err != nil {
		return err
	}

	err = validateServices(mainOptions, isFullMode)
	if err != nil {
		return err
	}

	err = validateServers(mainOptions, isFullMode)
	if err != nil {
		return err
	}

	err = validateRoutes(mainOptions, isFullMode)
	if err != nil {
		return err
	}

	err = validateMetrics(mainOptions, isFullMode)
	if err != nil {
		return err
	}

	return nil
}

func validateProviders(mainOptions Options) error {

	if mainOptions.Providers.File.Enabled && len(mainOptions.Providers.File.Paths) == 0 {
		return errors.New("the paths can't be empty for the file provider")
	}

	if mainOptions.Providers.Nacos.Config.Enabled {
		if len(mainOptions.Providers.Nacos.Config.Endpoints) == 0 {
			return errors.New("the endpoints can't be empty for the nacos config provider")
		}

		for _, endpoint := range mainOptions.Providers.Nacos.Config.Endpoints {
			_, err := url.Parse(endpoint)
			if err != nil {
				return fmt.Errorf("the endpoint '%s' is invalid for nacos config provider, error: %w", endpoint, err)
			}

			if !(strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://")) {
				return fmt.Errorf("the endpoint '%s' is invalid for nacos config provider.  It should start with http:// or https://", endpoint)
			}
		}

		if len(mainOptions.Providers.Nacos.Config.Files) == 0 {
			return errors.New("the files can't be empty for the nacos config provider")
		}
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

		for _, cidr := range serverOptions.TrustedCIDRS {
			_, _, err := net.ParseCIDR(cidr)
			if err != nil {
				msg := fmt.Sprintf("the cidr '%s' is invalid for server '%s'", cidr, serverID)
				structure := []string{"servers", serverID, "trusted_cidrs"}
				return newInvalidConfig(structure, cidr, msg)
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

	servers := map[string]*router.Router{}

	for serverID := range mainOptions.Servers {
		servers[serverID] = router.NewRouter()
	}

	for _, route := range mainOptions.Routes {
		if route.ServiceID == "" {
			msg := fmt.Sprintf("the 'service_id' can't be empty in the route '%s'", route.ID)
			structure := []string{"routes", route.ID, "service_id"}
			return newInvalidConfig(structure, "", msg)
		}

		if len(route.Paths) == 0 {
			msg := fmt.Sprintf("the paths can't be empty in the route '%s'", route.ID)
			structure := []string{"routes", route.ID, "paths"}
			return newInvalidConfig(structure, "", msg)
		}

		if !isFullMode {
			continue
		}

		if route.ServiceID[0] != '$' {
			if _, found := mainOptions.Services[route.ServiceID]; !found {
				msg := fmt.Sprintf("the service '%s' can't be found in the route '%s'", route.ServiceID, route.ID)
				structure := []string{"routes", route.ID, "service_id"}
				return newInvalidConfig(structure, "", msg)
			}
		}

		for _, serverID := range route.Servers {
			if _, found := mainOptions.Servers[serverID]; !found {
				msg := fmt.Sprintf("the server '%s' can't be found in the route '%s'", serverID, route.ID)
				structure := []string{"routes", route.ID, "servers"}
				return newInvalidConfig(structure, serverID, msg)
			}
		}

		for _, m := range route.Middlewares {
			if len(m.Use) > 0 {
				if _, found := mainOptions.Middlewares[m.Use]; !found {
					msg := fmt.Sprintf("the middleware '%s' can't be found in the route '%s'", m.Use, route.ID)
					structure := []string{"routes", route.ID, "middlewares"}
					return newInvalidConfig(structure, m.Use, msg)
				}
			}

			if len(m.Type) > 0 {
				hander := middleware.FindHandlerByType(m.Type)
				if hander == nil {
					return fmt.Errorf("the middleware '%s' can't be found in the route '%s'", m.Type, route.ID)
				}
			}
		}

		if len(route.Servers) == 0 {
			for _, router := range servers {
				err := addRoute(router, *route)
				if err != nil {
					return err
				}
			}
		} else if len(route.Servers) > 0 {
			for _, serverName := range route.Servers {
				router := servers[serverName]
				err := addRoute(router, *route)
				if err != nil {
					return err
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
		if hostname[0] != '$' && !strings.EqualFold("localhost", hostname) && !strings.EqualFold("[::1]", hostname) {
			_, found := mainOptions.Upstreams[hostname]
			if !found {
				if dnsResolver != nil && !mainOptions.Resolver.SkipTest {
					ips, err := dnsResolver.Lookup(context.Background(), hostname)
					if err != nil {
						return fmt.Errorf("fail to lookup host '%s' in the service '%s', error: %w", hostname, serviceID, err)
					}

					if len(ips) == 0 {
						return fmt.Errorf("fail to lookup host '%s' in the service '%s', error: no ip found", hostname, serviceID)
					}
				} else {
					ip := net.ParseIP(hostname)
					if !(IsValidDomain(hostname) || ip != nil) {
						return fmt.Errorf("the upstream '%s' can't be found in the service '%s'", hostname, serviceID)
					}
				}
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

func validateUpstreams(mainOptions Options, isFullMode bool) error {

	for upstreamID, opt := range mainOptions.Upstreams {

		if !isFullMode {
			continue
		}

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

		for _, target := range opt.Targets {
			addr := extractAddr(target.Target)
			if !strings.EqualFold("localhost", addr) && !strings.EqualFold("[::1]", addr) {
				if dnsResolver != nil && !mainOptions.Resolver.SkipTest {
					ips, err := dnsResolver.Lookup(context.Background(), addr)
					if err != nil {
						return fmt.Errorf("fail to lookup host '%s' in the upstream '%s', error: %w", addr, upstreamID, err)
					}

					if len(ips) == 0 {
						return fmt.Errorf("fail to lookup host '%s' in the upstream '%s', error: no ip found", addr, upstreamID)
					}
				}
			}
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

func extractAddr(addrport string) string {
	parts := strings.Split(addrport, ":")

	if len(parts) == 1 {
		return addrport
	}

	return parts[0]
}

func addRoute(r *router.Router, routeOptions RouteOptions) error {

	for _, path := range routeOptions.Paths {
		path = strings.TrimSpace(path)
		var nodeType router.NodeType

		switch {
		case strings.HasPrefix(path, "~*"):
			continue
		case strings.HasPrefix(path, "~"):
			continue
		case strings.HasPrefix(path, "="):
			nodeType = router.Exact
			path = strings.TrimSpace(path[1:])
			if len(path) == 0 {
				return fmt.Errorf("config: exact route can't be empty in route: '%s'", routeOptions.ID)
			}
		case strings.HasPrefix(path, "^~"):
			nodeType = router.PreferentialPrefix
			path = strings.TrimSpace(path[2:])
			if len(path) == 0 {
				return fmt.Errorf("config: prefix route can't be empty in route: '%s'", routeOptions.ID)
			}

		default:
			if !strings.HasPrefix(path, "/") {
				return fmt.Errorf("config: '%s' is invalid path. Path needs to begin with '/'", path)
			}
			nodeType = router.Prefix
		}

		if len(routeOptions.Methods) == 0 {
			for _, method := range router.HTTPMethods {
				err := r.Add(method, path, nodeType, func(c context.Context, ctx *app.RequestContext) {})
				if err != nil {
					return err
				}
			}
		}

		for _, method := range routeOptions.Methods {
			method := strings.ToUpper(method)
			if !router.IsValidHTTPMethod(method) {
				return fmt.Errorf("http method %s is not valid", method)
			}

			err := r.Add(method, path, nodeType, func(c context.Context, ctx *app.RequestContext) {})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
