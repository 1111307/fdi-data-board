package simple

import (
	"github.com/go-kratos/kratos/v2/transport"
)

const (
	KindSimple transport.Kind = "simple"
)

var _ transport.Transporter = &Transport{}

// Transport is a redis transport.
type Transport struct {
	endpoint  string
	operation string
}

// Kind returns the transport kind.
func (tr *Transport) Kind() transport.Kind {
	return KindSimple
}

// Endpoint returns the transport endpoint.
func (tr *Transport) Endpoint() string {
	return tr.endpoint
}

// Operation returns the transport operation.
func (tr *Transport) Operation() string {
	return tr.operation
}

// RequestHeader returns the request header.
func (tr *Transport) RequestHeader() transport.Header {
	return nil
}

// ReplyHeader returns the reply header.
func (tr *Transport) ReplyHeader() transport.Header {
	return nil
}
