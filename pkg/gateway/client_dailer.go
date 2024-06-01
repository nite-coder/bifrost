package gateway

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/cloudwego/hertz/pkg/common/errors"
	"github.com/cloudwego/hertz/pkg/network"
	"github.com/cloudwego/hertz/pkg/network/netpoll"
	cnetpoll "github.com/cloudwego/netpoll"
)

var errNotSupportTLS = errors.NewPublic("not support tls")

type dialer struct {
	cnetpoll.Dialer
}

func (d dialer) DialConnection(n, address string, timeout time.Duration, tlsConfig *tls.Config) (conn network.Conn, err error) {
	if tlsConfig != nil {
		// https
		return nil, errNotSupportTLS
	}
	c, err := d.Dialer.DialConnection(n, address, timeout)
	if err != nil {
		return nil, err
	}
	conn = &netpoll.Conn{Conn: c.(network.Conn)}
	return
}

func (d dialer) DialTimeout(network, address string, timeout time.Duration, tlsConfig *tls.Config) (conn net.Conn, err error) {
	if tlsConfig != nil {
		return nil, errNotSupportTLS
	}
	conn, err = d.Dialer.DialTimeout(network, address, timeout)
	return
}

func (d dialer) AddTLS(conn network.Conn, tlsConfig *tls.Config) (network.Conn, error) {
	return nil, errNotSupportTLS
}

func newDialer() network.Dialer {
	return dialer{Dialer: cnetpoll.NewDialer()}
}
