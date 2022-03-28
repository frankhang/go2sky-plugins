package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"
	"github.com/apache/thrift/lib/go/thrift"
	thrift_plugin "github.com/frankhang/go2sky-plugins/thrift"
	"github.com/frankhang/go2sky-plugins/thrift/test/pkg/gen-go/example"
	"log"
	"math/rand"
	"strings"
	"time"
)

type FormatDataImpl struct{}

const (
	LISTEN_HOST = "0.0.0.0"
	LISTEN_PORT = "8192"
	OAP_ADDR    = "localhost:11800"
	//OAP_ADDR = "114.116.113.17:30800"

	componentIDGOThriftServer = 5100
)

func main() {

	//这段代码放在某个初始化的地方，只执行一次。
	r, err := reporter.NewGRPCReporter(OAP_ADDR) //指定上报地址创建Reporter
	if err != nil {
		log.Println("NewGRPCReporter error: ", err)
	}
	defer r.Close()
	tracer, err := go2sky.NewTracer("Server2", go2sky.WithReporter(r), go2sky.WithSampler(1)) //指定采样率为1创建Tracer
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
	transportFactory := thrift.NewTTransportFactory()
	//protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	protocolFactory := thrift_plugin.NewTProtocolFactoryWrapper(thrift.NewTBinaryProtocolFactoryDefault(), nil, tracer) //将原来的ProtocolFactory传入thrift plugin的Wrapper

	server := thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)
	fmt.Println("Running at:", LISTEN_HOST+":"+LISTEN_PORT)
	if err := server.Serve(); err != nil {
		log.Fatalln("Error:", err)
	}
}

func (fdi *FormatDataImpl) DoTransfer(ctx context.Context, data *example.Data) (r *example.Data, err error) {
	return nil, nil
}

func (fdi *FormatDataImpl) DoFormat(ctx context.Context, data *example.Data) (r *example.Data, err error) {

	rand.Seed(time.Now().UnixNano())
	delay := rand.Intn(1000)
	fmt.Printf("Delay for %dms...\n", delay)
	time.Sleep(time.Duration(delay) * time.Millisecond)

	var rData string
	if delay > 700 { //设定30%的失败率，用于测试失败的情形
		err = errors.New("DoFormat doing err.")
	} else {
		rData = strings.ToUpper(data.Text)
	}
	r = &example.Data{
		Text: rData,
	}

	fmt.Println(r.Text)

	return
}

func (fdi *FormatDataImpl) DoBracket(ctx context.Context, data *example.Data) (r *example.Data, err error) {

	rand.Seed(time.Now().UnixNano())
	delay := rand.Intn(100)
	fmt.Printf("Delay for %dms...\n", delay)
	time.Sleep(time.Duration(delay) * time.Millisecond)

	r = &example.Data{
		Text: fmt.Sprintf("[%s]", data.Text),
	}

	fmt.Println(r.Text)

	return
}
