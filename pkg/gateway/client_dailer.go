package gateway

import (
	"context"
	"crypto/tls"
	"log/slog"
	"math/rand"
	"net"
	"time"

	"github.com/cloudwego/hertz/pkg/common/errors"
	"github.com/cloudwego/hertz/pkg/network"
	"github.com/cloudwego/hertz/pkg/network/netpoll"
	cnetpoll "github.com/cloudwego/netpoll"
	"github.com/rs/dnscache"
)

var errNotSupportTLS = errors.NewPublic("not support tls")

type httpDialer struct {
	cnetpoll.Dialer
	resolver dnscache.DNSResolver
}

func newHTTPDialer(resolver dnscache.DNSResolver) network.Dialer {
	return httpDialer{
		Dialer:   cnetpoll.NewDialer(),
		resolver: resolver,
	}
}

func (d httpDialer) DialConnection(n, address string, timeout time.Duration, tlsConfig *tls.Config) (conn network.Conn, err error) {
	if tlsConfig != nil {
		// https
		return nil, errNotSupportTLS
	}

	if d.resolver != nil {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := d.resolver.LookupHost(context.Background(), host)
		if err != nil {
			return nil, err
		}

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		randomIndex := r.Intn(len(ips))
		address = net.JoinHostPort(ips[randomIndex], port)
		slog.Debug("dns resolver info", "host", host, "ip", address)
	}

	c, err := d.Dialer.DialConnection(n, address, timeout)
	if err != nil {
		return nil, err
	}
	conn = &netpoll.Conn{Conn: c.(network.Conn)}
	return
}

func (d httpDialer) DialTimeout(network, address string, timeout time.Duration, tlsConfig *tls.Config) (conn net.Conn, err error) {
	if tlsConfig != nil {
		return nil, errNotSupportTLS
	}
	conn, err = d.Dialer.DialTimeout(network, address, timeout)
	return
}

func (d httpDialer) AddTLS(conn network.Conn, tlsConfig *tls.Config) (network.Conn, error) {
	return nil, errNotSupportTLS
}
