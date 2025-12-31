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
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/resolver"
	"github.com/nite-coder/bifrost/pkg/router"
	"github.com/nite-coder/bifrost/pkg/variable"
)

// ValidateConfig checks if the config's values are valid, but does not check if the config's value mapping is valid
func ValidateConfig(mainOptions Options, isFullMode bool) error {

	if dnsResolver == nil && !mainOptions.SkipResolver {
		var err error
		dnsResolver, err = resolver.NewResolver(resolver.Options{
			Servers: mainOptions.Resolver.Servers,
		})

		if err != nil {
			return err
		}
	}

	err := validateLogging(mainOptions.Logging)
	if err != nil {
		return err
	}

	err = validateResolver(mainOptions)
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

	err = validateMiddlewares(mainOptions, isFullMode)
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
		return errors.New("paths cannot be empty for file provider")
	}

	if mainOptions.Providers.Nacos.Config.Enabled {
		if len(mainOptions.Providers.Nacos.Config.Endpoints) == 0 {
			return errors.New("endpoints cannot be empty for Nacos config provider")
		}

		for _, endpoint := range mainOptions.Providers.Nacos.Config.Endpoints {
			_, err := url.Parse(endpoint)
			if err != nil {
				return fmt.Errorf("invalid endpoint '%s' for Nacos config provider: %w", endpoint, err)
			}

			if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
				return fmt.Errorf("invalid endpoint '%s' for Nacos config provider; must start with http:// or https://", endpoint)
			}
		}

		if len(mainOptions.Providers.Nacos.Config.Files) == 0 {
			return errors.New("files cannot be empty for Nacos config provider")
		}
	}

	if mainOptions.Providers.Nacos.Discovery.Enabled {
		if len(mainOptions.Providers.Nacos.Discovery.Endpoints) == 0 {
			return errors.New("endpoints cannot be empty for Nacos discovery provider")
		}

		for _, endpoint := range mainOptions.Providers.Nacos.Discovery.Endpoints {
			_, err := url.Parse(endpoint)
			if err != nil {
				return fmt.Errorf("invalid endpoint '%s' for Nacos discovery provider: %w", endpoint, err)
			}

			if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
				return fmt.Errorf("invalid endpoint '%s' for Nacos discovery provider; must start with http:// or https://", endpoint)
			}
		}
	}

	return nil
}

func validateLogging(opts LoggingOtions) error {

	level := strings.ToLower(opts.Level)
	switch level {
	case "", "debug", "info", "warn", "error":
	default:
		msg := fmt.Sprintf("unsupported logging level '%s'", level)
		structure := []string{"logging", "level"}
		return newInvalidConfig(structure, level, msg)
	}

	handler := strings.ToLower(opts.Handler)
	switch handler {
	case "text", "json", "":
	default:
		msg := fmt.Sprintf("unsupported logging handler '%s'", opts.Handler)
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
		return errors.New("service_name cannot be empty for tracing")
	}

	for _, propagator := range opts.Propagators {
		switch propagator {
		case "b3", "tracecontext", "baggage", "jaeger": // ok
		case "":
			return errors.New("propagator cannot be empty for tracing")
		default:
			return fmt.Errorf("unsupported propagator '%s' for tracing", propagator)
		}
	}

	return nil
}

func validateAccessLog(options map[string]AccessLogOptions) error {
	reIsVariable := regexp.MustCompile(`\$\w+(?:[._-]\w+)*`)

	for id, opt := range options {
		if opt.Template == "" {
			msg := fmt.Sprintf("template cannot be empty for access log '%s'", id)
			structure := []string{"access_logs", id, "template"}
			return newInvalidConfig(structure, opt.Template, msg)
		}

		if len(opt.TimeFormat) > 0 {
			_, err := time.Parse(opt.TimeFormat, time.Now().Format(opt.TimeFormat))
			if err != nil {
				msg := fmt.Sprintf("invalid time format '%s' for access log '%s'", opt.TimeFormat, id)
				structure := []string{"access_logs", id, "time_format"}
				return newInvalidConfig(structure, opt.TimeFormat, msg)
			}
		}

		if len(opt.Escape) > 0 {
			switch opt.Escape {
			case "json", "none", "default", "":
			default:
				msg := fmt.Sprintf("invalid escape '%s' for access log '%s'", opt.Escape, id)
				structure := []string{"access_logs", id, "escape"}
				return newInvalidConfig(structure, opt.Escape, msg)
			}
		}

		directives := reIsVariable.FindAllString(opt.Template, -1)

		for _, directive := range directives {
			if !variable.IsDirective(directive) {
				return fmt.Errorf("unsupported directive '%s' for access log '%s'", directive, id)
			}
		}
	}

	return nil
}

func validateMiddlewares(mainOptions Options, isFullMode bool) error {

	for middlewareID, middlewareOptions := range mainOptions.Middlewares {
		if len(middlewareOptions.Use) > 0 {
			return fmt.Errorf("middleware '%s' cannot run in 'use' mode for middleware ID: %s", middlewareOptions.Use, middlewareID)
		}

		if len(middlewareOptions.Type) == 0 {
			return fmt.Errorf("middleware type cannot be empty for middleware ID: %s", middlewareID)
		}

		if isFullMode {
			if len(middlewareOptions.Type) > 0 {
				hander := middleware.Factory(middlewareOptions.Type)
				if hander == nil {
					return fmt.Errorf("middleware '%s' not found for middleware ID: %s", middlewareOptions.Type, middlewareID)
				}
			}
		}
	}

	return nil
}

func validateServers(mainOptions Options, isFullMode bool) error {

	for serverID, serverOptions := range mainOptions.Servers {
		if serverOptions.Bind == "" {
			msg := "bind cannot be empty for server ID: " + serverID
			structure := []string{"servers", serverID, "bind"}
			return newInvalidConfig(structure, "", msg)
		}

		if len(serverOptions.TLS.CertPEM) > 0 {
			_, err := os.ReadFile(serverOptions.TLS.CertPEM)
			if err != nil {
				msg := "invalid cert PEM file for server ID: " + serverID

				if os.IsNotExist(err) {
					msg = "cert PEM file not found for server ID: " + serverID
				}

				structure := []string{"servers", serverID, "tls", "cert_pem"}
				return newInvalidConfig(structure, serverOptions.TLS.CertPEM, msg)
			}
		}

		if len(serverOptions.TLS.KeyPEM) > 0 {
			_, err := os.ReadFile(serverOptions.TLS.KeyPEM)
			if err != nil {
				msg := "invalid key PEM file for server ID: " + serverID

				if os.IsNotExist(err) {
					msg = "key PEM file not found for server ID: " + serverID
				}

				structure := []string{"servers", serverID, "tls", "key_pem"}
				return newInvalidConfig(structure, serverOptions.TLS.KeyPEM, msg)
			}
		}

		if serverOptions.AccessLogID != "" {
			if _, found := mainOptions.AccessLogs[serverOptions.AccessLogID]; !found {
				msg := fmt.Sprintf("access log '%s' not found for server ID: %s", serverOptions.AccessLogID, serverID)
				structure := []string{"servers", serverID, "access_log_id"}
				return newInvalidConfig(structure, serverOptions.AccessLogID, msg)
			}
		}

		for _, cidr := range serverOptions.TrustedCIDRS {
			_, _, err := net.ParseCIDR(cidr)
			if err != nil {
				msg := fmt.Sprintf("invalid CIDR '%s' for server ID: %s", cidr, serverID)
				structure := []string{"servers", serverID, "trusted_cidrs"}
				return newInvalidConfig(structure, cidr, msg)
			}
		}

		if isFullMode {
			for _, m := range serverOptions.Middlewares {
				if len(m.Use) > 0 {
					if _, found := mainOptions.Middlewares[m.Use]; !found {
						return fmt.Errorf("middleware '%s' not found for server ID: %s", m.Use, serverID)
					}
				}

				if len(m.Type) > 0 {
					hander := middleware.Factory(m.Type)
					if hander == nil {
						return fmt.Errorf("middleware '%s' not found for server ID: %s", m.Type, serverID)
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
		route.ServiceID = strings.TrimSpace(route.ServiceID)

		if route.ServiceID == "" {
			msg := "service_id cannot be empty for route ID: " + route.ID
			structure := []string{"routes", route.ID, "service_id"}
			return newInvalidConfig(structure, "", msg)
		}

		if len(route.Paths) == 0 {
			msg := "paths cannot be empty for route ID: " + route.ID
			structure := []string{"routes", route.ID, "paths"}
			return newInvalidConfig(structure, "", msg)
		}

		if !isFullMode {
			continue
		}

		if route.ServiceID[0] != '$' {
			if route.ServiceID == "_" {
				break
			}

			if _, found := mainOptions.Services[route.ServiceID]; !found {
				msg := fmt.Sprintf("service '%s' not found for route ID: %s", route.ServiceID, route.ID)
				structure := []string{"routes", route.ID, "service_id"}
				return newInvalidConfig(structure, "", msg)
			}
		}

		for _, serverID := range route.Servers {
			if _, found := mainOptions.Servers[serverID]; !found {
				msg := fmt.Sprintf("server '%s' not found for route ID: %s", serverID, route.ID)
				structure := []string{"routes", route.ID, "servers"}
				return newInvalidConfig(structure, serverID, msg)
			}
		}

		for _, m := range route.Middlewares {
			if len(m.Use) > 0 {
				if _, found := mainOptions.Middlewares[m.Use]; !found {
					msg := fmt.Sprintf("middleware '%s' not found for route ID: %s", m.Use, route.ID)
					structure := []string{"routes", route.ID, "middlewares"}
					return newInvalidConfig(structure, m.Use, msg)
				}
			}

			if len(m.Type) > 0 {
				hander := middleware.Factory(m.Type)
				if hander == nil {
					return fmt.Errorf("middleware '%s' not found for route ID: %s", m.Type, route.ID)
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

		addr, err := url.Parse(service.URL)
		if err != nil {
			return err
		}

		hostname := addr.Hostname()

		// validate
		if len(hostname) == 0 {
			return fmt.Errorf("invalid host in service URL for service ID: %s", serviceID)
		}

		// exist upstream
		if hostname[0] != '$' && !strings.EqualFold("localhost", hostname) && !strings.EqualFold("[::1]", hostname) {
			_, found := mainOptions.Upstreams[hostname]
			if !found {
				if dnsResolver != nil && !mainOptions.SkipResolver {
					ips, err := dnsResolver.Lookup(context.Background(), hostname)
					if err != nil {
						return fmt.Errorf("failed to lookup host '%s' for service ID '%s': %w", hostname, serviceID, err)
					}

					if len(ips) == 0 {
						return fmt.Errorf("failed to lookup host '%s' for service ID '%s': no IP found", hostname, serviceID)
					}
				} else {
					ip := net.ParseIP(hostname)
					if !IsValidDomain(hostname) && ip == nil {
						return fmt.Errorf("upstream '%s' not found for service ID: %s", hostname, serviceID)
					}
				}
			}
		}

		for _, m := range service.Middlewares {
			if len(m.Use) > 0 {
				if _, found := mainOptions.Middlewares[m.Use]; !found {
					return fmt.Errorf("middleware '%s' not found for service ID: %s", m.Use, serviceID)
				}
			}

			if len(m.Type) > 0 {
				hander := middleware.Factory(m.Type)
				if hander == nil {
					return fmt.Errorf("middleware '%s' not found for service ID: %s", m.Type, serviceID)
				}
			}
		}
	}

	return nil
}

func validateUpstreams(mainOptions Options, isFullMode bool) error {

	for upstreamID, upstreamOptions := range mainOptions.Upstreams {

		if !isFullMode {
			continue
		}

		if upstreamID[0] == '$' {
			msg := fmt.Sprintf("upstream ID '%s' cannot start with '$'", upstreamID)
			structure := []string{"upstreams", upstreamID}
			return newInvalidConfig(structure, "", msg)
		}

		factory := balancer.Factory(upstreamOptions.Balancer.Type)
		if factory == nil && upstreamOptions.Balancer.Type == "" {
		} else if factory == nil {
			msg := fmt.Sprintf("unsupported balancer strategy '%s' for upstream ID: %s", upstreamOptions.Balancer, upstreamID)
			structure := []string{"upstreams", upstreamID, "strategy"}
			return newInvalidConfig(structure, upstreamOptions.Balancer, msg)
		}

		switch upstreamOptions.Discovery.Type {
		case "dns":
			if !mainOptions.Providers.DNS.Enabled {
				return fmt.Errorf("DNS provider is disabled for upstream ID: %s", upstreamID)
			}

			if upstreamOptions.Discovery.Name == "" {
				return fmt.Errorf("discovery name cannot be empty for upstream ID: %s", upstreamID)
			}
		case "nacos":
			if !mainOptions.Providers.Nacos.Discovery.Enabled {
				return fmt.Errorf("nacos service discovery provider is disabled for upstream ID: %s", upstreamID)
			}

			if upstreamOptions.Discovery.Name == "" {
				return fmt.Errorf("discovery name cannot be empty for upstream ID: %s", upstreamID)
			}
		case "k8s":
			if !mainOptions.Providers.K8S.Enabled {
				return fmt.Errorf("K8s service discovery provider is disabled for upstream ID: %s", upstreamID)
			}

			if upstreamOptions.Discovery.Name == "" {
				return fmt.Errorf("discovery name cannot be empty for upstream ID: %s", upstreamID)
			}
		case "":
			if upstreamOptions.Discovery.Type == "" && len(upstreamOptions.Targets) == 0 {
				return fmt.Errorf("targets cannot be empty for upstream ID: %s", upstreamID)
			}
		default:
			msg := fmt.Sprintf("unsupported discovery type '%s' for upstream ID: %s", upstreamOptions.Discovery.Type, upstreamID)
			structure := []string{"upstreams", upstreamID, "discovery", "type"}
			return newInvalidConfig(structure, upstreamOptions.Discovery.Type, msg)
		}

		for _, target := range upstreamOptions.Targets {
			addr := extractAddr(target.Target)
			if !strings.EqualFold("localhost", addr) && !strings.EqualFold("[::1]", addr) {
				if dnsResolver != nil && !mainOptions.SkipResolver {
					ips, err := dnsResolver.Lookup(context.Background(), addr)
					if err != nil {
						return fmt.Errorf("failed to lookup host '%s' for upstream ID '%s': %w", addr, upstreamID, err)
					}

					if len(ips) == 0 {
						return fmt.Errorf("failed to lookup host '%s' for upstream ID '%s': no IP found", addr, upstreamID)
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
			return errors.New("server_id cannot be empty for Prometheus")
		}

		if !isFullMode {
			return nil
		}

		_, found := options.Servers[options.Metrics.Prometheus.ServerID]
		if !found {
			msg := "server_id '" + options.Metrics.Prometheus.ServerID + "' not found for prometheus"
			structure := []string{"metrics", "prometheus", "server_id"}
			return newInvalidConfig(structure, options.Metrics.Prometheus.ServerID, msg)
		}
	}

	return nil
}

func validateResolver(options Options) error {

	if len(options.Resolver.Hostsfile) > 0 {
		if _, err := os.Stat(options.Resolver.Hostsfile); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("hosts file '%s' not found", options.Resolver.Hostsfile)
		}
	}

	if len(options.Resolver.Order) > 0 {
		for _, order := range options.Resolver.Order {
			order = strings.TrimSpace(order)
			switch strings.ToLower(order) {
			case "last", "a", "cname":
			default:
				msg := fmt.Sprintf("unsupported resolver order '%s'", order)
				structure := []string{"resolver", "order"}
				return newInvalidConfig(structure, order, msg)
			}
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
				return fmt.Errorf("config: exact route path cannot be empty for route ID: %s", routeOptions.ID)
			}
		case strings.HasPrefix(path, "^~"):
			nodeType = router.PreferentialPrefix
			path = strings.TrimSpace(path[2:])
			if len(path) == 0 {
				return fmt.Errorf("config: prefix route path cannot be empty for route ID: %s", routeOptions.ID)
			}

		default:
			if !strings.HasPrefix(path, "/") {
				return fmt.Errorf("config: invalid path '%s'; must begin with '/'", path)
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
				return fmt.Errorf("HTTP method %s is not valid", method)
			}

			err := r.Add(method, path, nodeType, func(c context.Context, ctx *app.RequestContext) {})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
