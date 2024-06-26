package gateway

import (
	"context"
	"crypto/tls"
	"log/slog"
	"math/rand"
	"net"
	"time"

	"github.com/cloudwego/hertz/pkg/network"
	"github.com/cloudwego/hertz/pkg/network/netpoll"
	"github.com/cloudwego/hertz/pkg/network/standard"
	"github.com/rs/dnscache"
)

type httpDialer struct {
	dialer   network.Dialer
	resolver dnscache.DNSResolver
	random   *rand.Rand
}

func newHTTPDialer(resolver dnscache.DNSResolver) network.Dialer {
	return &httpDialer{
		dialer:   netpoll.NewDialer(),
		resolver: resolver,
		random:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (d *httpDialer) DialConnection(n, address string, timeout time.Duration, tlsConfig *tls.Config) (conn network.Conn, err error) {
	if d.resolver != nil {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := d.resolver.LookupHost(context.Background(), host)
		if err != nil {
			return nil, err
		}

		randomIndex := d.random.Intn(len(ips))
		address = net.JoinHostPort(ips[randomIndex], port)
		slog.Debug("http dns resolver info", "host", host, "ip", address)
	}

	return d.dialer.DialConnection(n, address, timeout, tlsConfig)
}

func (d *httpDialer) DialTimeout(network, address string, timeout time.Duration, tlsConfig *tls.Config) (conn net.Conn, err error) {
	return d.dialer.DialConnection(network, address, timeout, tlsConfig)
}

func (d *httpDialer) AddTLS(conn network.Conn, tlsConfig *tls.Config) (network.Conn, error) {
	return d.dialer.AddTLS(conn, tlsConfig)
}

type httpsDialer struct {
	dialer   network.Dialer
	resolver dnscache.DNSResolver
	random   *rand.Rand
}

func newHTTPSDialer(resolver dnscache.DNSResolver) network.Dialer {
	return &httpsDialer{
		dialer:   standard.NewDialer(),
		resolver: resolver,
		random:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (d *httpsDialer) DialConnection(n, address string, timeout time.Duration, tlsConfig *tls.Config) (conn network.Conn, err error) {
	if d.resolver != nil {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := d.resolver.LookupHost(context.Background(), host)
		if err != nil {
			return nil, err
		}

		randomIndex := d.random.Intn(len(ips))
		address = net.JoinHostPort(ips[randomIndex], port)
		slog.Debug("https dns resolver info", "host", host, "ip", address)
	}

	return d.dialer.DialConnection(n, address, timeout, tlsConfig)
}

func (d *httpsDialer) DialTimeout(network, address string, timeout time.Duration, tlsConfig *tls.Config) (conn net.Conn, err error) {
	return d.dialer.DialConnection(network, address, timeout, tlsConfig)
}

func (d *httpsDialer) AddTLS(conn network.Conn, tlsConfig *tls.Config) (network.Conn, error) {
	return d.dialer.AddTLS(conn, tlsConfig)
}
