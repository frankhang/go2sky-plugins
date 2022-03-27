package main

import (
	"context"
	"fmt"
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"
	"github.com/apache/thrift/lib/go/thrift"
	thrift_plugin "github.com/frankhang/go2sky-plugins/thrift"
	"github.com/frankhang/go2sky-plugins/thrift/test/ThriftDemo/gen-go/example"
	"log"
	"net"
	"time"
)

const (
	REQ_HOST = "localhost"
	REQ_PORT = "8191"
	OAP_ADDR = "localhost:11800"
	//OAP_ADDR = "114.116.113.17:30800"

	componentIDGOThriftClient = 5101
)

var defaultCtx = context.Background()

func main() {

	//这段代码放在某个初始化的地方，只执行一次。
	r, err := reporter.NewGRPCReporter(OAP_ADDR) //指定上报地址创建Reporter
	if err != nil {
		log.Println("NewGRPCReporter error: ", err)
	}
	defer r.Close()
	tracer, err := go2sky.NewTracer("client", go2sky.WithReporter(r), go2sky.WithSampler(1)) //创建采样率为1的Tracer
	if err != nil {
		log.Println("NewTracer error:", err)
	}

	/////////////////////

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

	if err := transport.Open(); err != nil {
		log.Fatalln("Error opening:", REQ_HOST+":"+REQ_PORT)
	}
	defer transport.Close()
	client := example.NewFormatDataClientFactory(transport, protocolFactory)

	data := example.Data{
		Text: "hello,world!",
	}
	//callDoTransfer(tracer, defaultCtx, reqAddr, &data, client)
	//callDoTransfer(tracer, defaultCtx, reqAddr, &data, client)

	for {
		//fmt.Println("routing3...")
		callDoTransfer(&data, client)

		time.Sleep(time.Duration(2) * time.Second)

	}

}

func callDoTransfer(data *example.Data, client *example.FormatDataClient) {

	r, err := client.DoTransfer(defaultCtx, data)
	if err != nil {
		fmt.Println("callDoTransfer error.", err)

	} else {

		fmt.Println(r.Text)
	}

}
