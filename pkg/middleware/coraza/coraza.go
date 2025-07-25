package coroza

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/cloudwego/hertz/pkg/app"
	coreruleset "github.com/corazawaf/coraza-coreruleset/v4"
	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/nite-coder/blackbear/pkg/cast"
	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	bifrostWAFCoreRulesetHits *prom.CounterVec
)

const (
	labelServerID     = "server_id"
	labelRuleID       = "rule_id"
	labelClientIP     = "client_ip"
	labelMethod       = "method"
	labelPath         = "path"
	unknownLabelValue = "unknown"
)

type CorazaMiddleware struct {
	options *Options
	waf     coraza.WAF
}
type Options struct {
	Directives               string   `mapstructure:"directives"`
	RejectedHTTPContentType  string   `mapstructure:"rejected_http_content_type"`
	RejectedHTTPResponseBody string   `mapstructure:"rejected_http_response_body"`
	IPAllowList              []string `mapstructure:"ip_allow_list"`
	RejectedHTTPStatusCode   int      `mapstructure:"rejected_http_status_code"`
}

func NewMiddleware(options Options) (*CorazaMiddleware, error) {
	if options.Directives == "" {
		options.Directives = `
			Include @coraza.conf-recommended
			Include @crs-setup.conf.example
			Include @owasp_crs/*.conf
			`
	}
	if options.RejectedHTTPStatusCode == 0 {
		options.RejectedHTTPStatusCode = 403
	}
	waf, err := coraza.NewWAF(
		coraza.NewWAFConfig().
			WithRootFS(coreruleset.FS).
			WithDirectives(options.Directives),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Coraza WAF: %w", err)
	}
	return &CorazaMiddleware{
		options: &options,
		waf:     waf,
	}, nil
}
func (m *CorazaMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := log.FromContext(ctx)
	clientIP := c.ClientIP()
	for _, allowIP := range m.options.IPAllowList {
		_, ipNet, err := net.ParseCIDR(allowIP)
		if err != nil {
			// If not CIDR, check exact IP match
			if clientIP == allowIP {
				c.Next(ctx)
				return
			}
			continue
		}
		// Parse client IP
		ip := net.ParseIP(clientIP)
		if ip == nil {
			continue
		}
		// Check if client IP is in CIDR range
		if ipNet.Contains(ip) {
			c.Next(ctx)
			return
		}
	}
	tx := m.waf.NewTransaction()
	defer tx.Close()
	// Process Connection
	peerAddr := c.RemoteAddr()
	targetHost, targetPort, _ := net.SplitHostPort(peerAddr.String())
	port, _ := cast.ToInt(targetPort)
	tx.ProcessConnection(targetHost, port, "", 0)
	// Process URI
	method := variable.GetString(variable.HTTPRequestMethod, c)
	reqURI := variable.GetString(variable.HTTPRequestURI, c)
	proto := variable.GetString(variable.HTTPRequestProtocol, c)
	tx.ProcessURI(reqURI, method, proto)
	// Add Request Headers
	c.Request.Header.VisitAll(func(key, value []byte) {
		k := cast.B2S(key)
		val := cast.B2S(value)
		tx.AddRequestHeader(k, val)
	})
	// Process Phase 1 - Request Headers
	it := tx.ProcessRequestHeaders()
	m.processInterruption(ctx, c, tx, it)
	// Read and Process Body if exists
	if tx.IsRequestBodyAccessible() {
		if !c.Request.IsBodyStream() && len(c.Request.Body()) > 0 {
			it, _, err := tx.WriteRequestBody(c.Request.Body())
			if err != nil {
				logger.Warn("coraza: failed to write request body", "error", err)
				return
			}
			m.processInterruption(ctx, c, tx, it)
		}
	}
	it, err := tx.ProcessRequestBody()
	if err != nil {
		logger.Warn("coraza: failed to process request body", "error", err)
		return
	}
	m.processInterruption(ctx, c, tx, it)
	m.log(ctx, c, tx)
	c.Next(ctx)
}
func (m *CorazaMiddleware) processInterruption(ctx context.Context, c *app.RequestContext, tx types.Transaction, it *types.Interruption) {
	if it == nil {
		return
	}
	if it.Action == "deny" {
		m.log(ctx, c, tx)
		c.SetStatusCode(m.options.RejectedHTTPStatusCode)
		if len(m.options.RejectedHTTPContentType) > 0 {
			c.Response.Header.Set("Content-Type", m.options.RejectedHTTPContentType)
		}
		if len(m.options.RejectedHTTPResponseBody) > 0 {
			c.Response.SetBody([]byte(m.options.RejectedHTTPResponseBody))
		}
		c.Abort()
		return
	}
}
func (m *CorazaMiddleware) log(ctx context.Context, c *app.RequestContext, tx types.Transaction) {
	logger := log.FromContext(ctx)
	matchedRules := tx.MatchedRules()
	if len(matchedRules) > 0 {
		for _, rule := range tx.MatchedRules() {
			msg := rule.Message()
			if len(msg) > 0 {
				ruleID := rule.Rule().ID()
				ruleIDStr, _ := cast.ToString(ruleID)
				serverID := variable.GetString(variable.ServerID, c)
				logger.WarnContext(ctx, "forbidden by WAF",
					"rule_id", ruleID,
					"message", msg,
					"client_ip", rule.ClientIPAddress(),
					"method", variable.GetString(variable.HTTPRequestMethod, c),
					"full_uri", rule.URI(),
					"data", rule.Data(),
				)
				// rule 949110 It is only used to calculate the final score, so it does not count as an attack.
				if ruleID == 949110 {
					continue
				}
				labels := make(prom.Labels, 5)
				labels[labelServerID] = defaultValIfEmpty(serverID, unknownLabelValue)
				labels[labelRuleID] = defaultValIfEmpty(ruleIDStr, unknownLabelValue)
				labels[labelClientIP] = defaultValIfEmpty(rule.ClientIPAddress(), unknownLabelValue)
				method := variable.GetString(variable.HTTPRequestMethod, c)
				labels[labelMethod] = defaultValIfEmpty(method, unknownLabelValue)
				path := variable.GetString(variable.HTTPRoute, c)
				if path == "" {
					path = variable.GetString(variable.HTTPRequestPath, c)
				}
				labels[labelPath] = defaultValIfEmpty(path, unknownLabelValue)
				_ = counterAdd(bifrostWAFCoreRulesetHits, 1, labels)
			}
		}
	}
}
func init() {
	bifrostWAFCoreRulesetHits = prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_waf_core_ruleset_hits",
			Help: "Number of WAF Core Ruleset hits",
		},
		[]string{"server_id", "method", "path", "rule_id", "client_ip"},
	)
	prom.MustRegister(bifrostWAFCoreRulesetHits)
	_ = middleware.Register([]string{"coraza"}, func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("coraza middleware params is empty or invalid")
		}
		options := Options{}
		decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
			WeaklyTypedInput: true,
			Result:           &options,
		})
		err := decoder.Decode(params)
		if err != nil {
			return nil, fmt.Errorf("coraza middleware params is invalid: %w", err)
		}
		m, err := NewMiddleware(options)
		if err != nil {
			return nil, err
		}
		return m.ServeHTTP, nil
	})
}
func defaultValIfEmpty(val, def string) string {
	if val == "" {
		return def
	}
	return val
}
func counterAdd(counterVec *prom.CounterVec, value int, labels prom.Labels) error {
	counter, err := counterVec.GetMetricWith(labels)
	if err != nil {
		return err
	}
	counter.Add(float64(value))
	return nil
}
