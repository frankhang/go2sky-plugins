#Golang thrift plugin for skywalking 使用指南

* 依赖

```
impport (
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"
	thrift_plugin "github.com/frankhang/go2sky-plugins/thrift"
)
```

thrift_plugin 下来后，修改其中的go.mod文件，将
```
replace github.com/apache/thrift/lib/go/thrift =>
```

指向自己提前装好的thrift0.13.0库的位置。


* 程序初始化的时候调用以下代码（仅调用一次,包括thrift客户方和服务方程序）


```
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

```

* Thrift服务方初始化代码做修改，举例如下：

```
	handler := &FormatDataImpl{}
	processor := example.NewFormatDataProcessor(handler)
	serverTransport, err := thrift.NewTServerSocket(LISTEN_HOST + ":" + LISTEN_PORT)
	if err != nil {
		log.Fatalln("Error:", err)
	}
	transportFactory := thrift.NewTTransportFactory()
	//如下，将thrift创建的ProtocolFactory传入本plugin对应的Wrapper
	protocolFactory := thrift_plugin.NewTProtocolFactoryWrapper(thrift.NewTBinaryProtocolFactoryDefault(), nil, tracer) 
	server := thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)
	server.Serve()
```

* Thrift客户方初始化代码做修改，举例如下：

```
	reqAddr := net.JoinHostPort(REQ_HOST, REQ_PORT)
	tSocket, err := thrift.NewTSocket(reqAddr)
	//如下，将thrift创建的TransportFactory传入本plugin对应的Wrapper
	transportFactory := thrift_plugin.NewTTransportFactoryWrapper(thrift.NewTTransportFactory())

	transport, _ := transportFactory.GetTransport(tSocket)

	//如下，将thrift创建的ProtocolFactory传入本plugin对应的Wrapper
	protocolFactory := thrift_plugin.NewTProtocolFactoryWrapper(thrift.NewTBinaryProtocolFactoryDefault(), transportFactory, tracer) 
	
	transport.Open()
	client := example.NewFormatDataClientFactory(transport, protocolFactory)



```

#Enjoy



