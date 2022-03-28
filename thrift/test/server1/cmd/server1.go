package main

import (
	"context"
	"fmt"
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"
	"github.com/apache/thrift/lib/go/thrift"
	thrift_plugin "github.com/frankhang/go2sky-plugins/thrift"
	"github.com/frankhang/go2sky-plugins/thrift/test/pkg/gen-go/example"
	"log"
	"net"
)

type FormatDataImpl struct{}

const (
	LISTEN_HOST = "0.0.0.0"
	LISTEN_PORT = "8191"

	REQ_HOST = "localhost"
	REQ_PORT = "8192"

	OAP_ADDR = "localhost:11800"
	//OAP_ADDR = "114.116.113.17:30800"

	componentIDGOThriftServer = 5100
	componentIDGOThriftClient = 5101
)

var (
	defaultCtx = context.Background()

	client *example.FormatDataClient
)

func main() {

	//这段代码放在某个初始化的地方，只执行一次。
	r, err := reporter.NewGRPCReporter(OAP_ADDR) //指定上报地址创建Reporter
	if err != nil {
		log.Println("NewGRPCReporter error: ", err)
	}
	defer r.Close()
	tracer, err := go2sky.NewTracer("Server1", go2sky.WithReporter(r), go2sky.WithSampler(1)) //指定采样率为1创建Tracer
	if err != nil {
		log.Println("NewTracer error:", err)
	}

	go2sky.SetGlobalTracer(tracer)

	/////////////////////

	//DefaultCtx := context.Background()
	handler := &FormatDataImpl{}
	processor := example.NewFormatDataProcessor(handler)
	serverTransport, err := thrift.NewTServerSocket(LISTEN_HOST + ":" + LISTEN_PORT)
	if err != nil {
		log.Fatalln("Error:", err)
	}
	reqAddr := net.JoinHostPort(REQ_HOST, REQ_PORT)
	tSocket, err := thrift.NewTSocket(reqAddr)
	if err != nil {
		log.Fatalln("tSocket error:", err)
	}

	//transportFactory := thrift.NewTTransportFactory()
	transportFactory := thrift_plugin.NewTTransportFactoryWrapper(thrift.NewTTransportFactory())
	transport, _ := transportFactory.GetTransport(tSocket)
	//protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	protocolFactory := thrift_plugin.NewTProtocolFactoryWrapper(thrift.NewTBinaryProtocolFactoryDefault(), transportFactory, tracer) //将原来的ProtocolFactory传入thrift plugin的Wrapper

	server := thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)

	if err := transport.Open(); err != nil {
		log.Fatalln("Error opening:", REQ_HOST+":"+REQ_PORT)
	}
	client = example.NewFormatDataClientFactory(transport, protocolFactory)

	fmt.Println("Running at:", LISTEN_HOST+":"+LISTEN_PORT)
	if err := server.Serve(); err != nil {
		log.Fatalln("Error:", err)
	}
}
func (fdi *FormatDataImpl) DoTransfer(ctx context.Context, data *example.Data) (r *example.Data, err error) {

	if r, err = callDoformat(data, client); err != nil {
		fmt.Printf("callDoformat error: %s", err.Error())
	} else {
		r, err = callDobracket(r, client)
	}

	return
}

func (fdi *FormatDataImpl) DoFormat(ctx context.Context, data *example.Data) (r *example.Data, err error) {
	return nil, nil
}

func (fdi *FormatDataImpl) DoBracket(ctx context.Context, data *example.Data) (r *example.Data, err error) {
	return nil, nil
}

func callDoformat(data *example.Data, client *example.FormatDataClient) (r *example.Data, err error) {

	//var errMsg string
	r, err = client.DoFormat(defaultCtx, data)
	if err != nil {
		log.Println("callDoformat error.", err)
	} else {
		fmt.Println(r.Text)
	}

	return

}

func callDobracket(data *example.Data, client *example.FormatDataClient) (r *example.Data, err error) {

	//var errMsg string
	r, err = client.DoBracket(defaultCtx, data)
	if err != nil {
		log.Println("callDoBracket error.", err)
	} else {
		fmt.Println(r.Text)
	}

	return

}
