package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// ValidateMapping checks if the config's value mapping is valid.  For example, the server in route must be finded in the servers
func ValidateMapping(mainOpts Options) error {
	if len(mainOpts.Servers) == 0 {
		return errors.New("no server found")
	}

	if len(mainOpts.Routes) == 0 {
		return errors.New("no route found")
	}

	for routeID, route := range mainOpts.Routes {
		for _, serverID := range route.Servers {
			if _, found := mainOpts.Servers[serverID]; !found {
				return fmt.Errorf("the server '%s' can't be found in the route '%s'", serverID, routeID)
			}
		}
	}

	return nil

}

// ValidateConfig checks if the config's values are valid, but does not check if the config's value mapping is valid
func ValidateConfig(mainOpts Options) error {

	err := validateLogging(mainOpts.Logging)
	if err != nil {
		return err
	}

	err = validateAccessLog(mainOpts.AccessLogs)
	if err != nil {
		return err
	}

	err = validateServers(mainOpts.Servers)
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
	for id, opt := range options {
		if !opt.Enabled {
			continue
		}

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
	}

	return nil
}

func validateServers(options map[string]ServerOptions) error {
	for id, opt := range options {
		if opt.Bind == "" {
			msg := fmt.Sprintf("the bind can't be empty for server '%s'", id)
			fullpath := []string{"servers", id, "bind"}
			return newInvalidConfig(fullpath, "", msg)
		}

		if len(opt.TLS.CertPEM) > 0 {
			_, err := os.ReadFile(opt.TLS.CertPEM)
			if err != nil {
				msg := fmt.Sprintf("the cert pem file is invalid for server '%s'", id)

				if os.IsNotExist(err) {
					msg = fmt.Sprintf("the cert pem file doesn't exist for server '%s'", id)
				}

				fullpath := []string{"servers", id, "tls", "cert_pem"}
				return newInvalidConfig(fullpath, opt.TLS.CertPEM, msg)
			}
		}

		if len(opt.TLS.KeyPEM) > 0 {
			_, err := os.ReadFile(opt.TLS.KeyPEM)
			if err != nil {
				msg := fmt.Sprintf("the key pem file is invalid for server '%s'", id)

				if os.IsNotExist(err) {
					msg = fmt.Sprintf("the key pem file doesn't exist for server '%s'", id)
				}

				fullpath := []string{"servers", id, "tls", "key_pem"}
				return newInvalidConfig(fullpath, opt.TLS.KeyPEM, msg)
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
