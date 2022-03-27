package thrift

import (
	"github.com/SkyAPM/go2sky"
	"github.com/apache/thrift/lib/go/thrift"
)

type TTransportFactoryWrapper struct {
	thrift.TTransportFactory
	tracer  *go2sky.Tracer
	reqAddr string
}

func NewTTransportFactoryWrapper(tf thrift.TTransportFactory) *TTransportFactoryWrapper {
	tfw := &TTransportFactoryWrapper{}
	tfw.TTransportFactory = tf
	return tfw
}

func (tfw *TTransportFactoryWrapper) GetTransport(s thrift.TTransport) (thrift.TTransport, error) {
	if socket, ok := s.(*thrift.TSocket); ok {
		tfw.reqAddr = socket.Addr().String()
	} else {
		tfw.reqAddr = "UNKNOWN"
	}

	return tfw.TTransportFactory.GetTransport(s)
}
