package thrift

import (
	"context"
	"fmt"
	"github.com/SkyAPM/go2sky"
	"github.com/apache/thrift/lib/go/thrift"
	"github.com/timandy/routine"
	v3 "skywalking.apache.org/repo/goapi/collect/language/agent/v3"
	"time"
)

const (
	SW_MAGIC_FIELD            = "SW_MAGIC_FIELD" // Field Name
	SW_MAGIC_FIELD_ID         = 8888             // Field ID, a magic number
	componentIDGOThriftServer = 5100
	componentIDGOThriftClient = 5101
)

type TProtocolWrapper struct {
	thrift.TProtocol
	tracer                       *go2sky.Tracer
	header                       map[string]string
	operationName                string
	entrySpan                    go2sky.Span
	span                         go2sky.Span
	reqAddr                      string
	isClient, isServer           bool
	headerReaded, headerInjected bool
	ignoreField                  bool
}

type TProtocolFactoryWrapper struct {
	thrift.TProtocolFactory
	tracer  *go2sky.Tracer
	reqAddr string
}

func newTProtocolWrapper(p thrift.TProtocol, tracer *go2sky.Tracer) *TProtocolWrapper {
	pw := &TProtocolWrapper{}
	pw.TProtocol = p
	pw.tracer = tracer
	return pw
}

func NewTProtocolFactoryWrapper(pf thrift.TProtocolFactory, tfw *TTransportFactoryWrapper, tracer *go2sky.Tracer) *TProtocolFactoryWrapper {
	pfw := &TProtocolFactoryWrapper{}
	pfw.TProtocolFactory = pf
	pfw.tracer = tracer
	if tfw != nil {
		pfw.reqAddr = tfw.reqAddr
	}
	return pfw
}

func (pfw *TProtocolFactoryWrapper) GetProtocol(t thrift.TTransport) thrift.TProtocol {
	p := pfw.TProtocolFactory.GetProtocol(t)

	pw := newTProtocolWrapper(p, pfw.tracer)
	pw.reqAddr = pfw.reqAddr
	return pw
}

func (pw *TProtocolWrapper) ReadMessageBegin() (name string, typeId thrift.TMessageType, seqid int32, err error) {
	name, typeId, seqid, err = pw.TProtocol.ReadMessageBegin()

	pw.operationName = name

	if typeId == thrift.CALL { //server side
		pw.isServer = true
		pw.header = make(map[string]string)
		pw.headerReaded = false
		//fmt.Printf("[%s] ReadMessageBegin: %v, %d, %p-%d\n", pw.operationName, routine.Goid(), seqid, pw, len(pw.header))

	} else { //client side echo
		//fmt.Printf("[%s] ReadMessageBegin2: %v, %d, %p-%d\n", pw.operationName, routine.Goid(), seqid, pw, len(pw.header))
		_, pw.span, _ = recoverFromContext()
		//fmt.Printf("ReadMessageBegin2 span [%s] : %p， %d\n", pw.operationName, pw.span, typeId)
		if pw.span != nil {
			if err != nil || typeId == thrift.EXCEPTION || typeId == thrift.INVALID_TMESSAGE_TYPE {
				//fmt.Printf("AAAAAAAA\n")
				pw.span.Error(time.Now(), fmt.Sprintf("%s error %d %s", name, typeId, err)) //错误上报，第二个参数开始（可选）表示错误日志。
				pw.span.Tag(go2sky.TagStatusCode, "1")                                      //错误代码上报，可选。
			} else {
				//pw.span.Log(time.Now(), name+" suc")   //日志上报，可选。
				pw.span.Tag(go2sky.TagStatusCode, "0") //错误代码上报，可选。
			}
			pw.span.End()
		}

	}
	return
}
func (pw *TProtocolWrapper) ReadMessageEnd() (err error) {
	if pw.isServer { //server side

		err = pw.TProtocol.ReadMessageEnd()
		if err != nil {
			fmt.Printf("[%s] ReadMessageEnd error: %v\n", pw.operationName, err)
			return
		}

		entrySpan, entryCtx, e2 := pw.tracer.CreateEntrySpan(context.Background(), pw.operationName, func(key string) (string, error) {
			return pw.header[key], nil
		})
		if e2 != nil {
			fmt.Println("CreateEntrySpan error: ", e2)
			return
		}
		//fmt.Printf("[%s] ReadMessageEnd: %v, %p, %p-%d\n", pw.operationName, routine.Goid(), entryCtx, pw, len(pw.header))

		//defer entrySpan.End()
		pw.entrySpan = entrySpan

		storeToContext(entryCtx, nil, entrySpan)

		if entrySpan != nil {
			entrySpan.SetComponent(componentIDGOThriftServer)
			entrySpan.SetSpanLayer(v3.SpanLayer_RPCFramework)
			entrySpan.SetOperationName(pw.operationName)
		}
	} else {
		//fmt.Printf("[%s] ReadMessageEnd2: %v, %p-%d\n", pw.operationName, routine.Goid(), pw, len(pw.header))

	}
	return
}

func (pw *TProtocolWrapper) ReadFieldBegin() (name string, typeId thrift.TType, id int16, err error) {
	p := pw.TProtocol
	name, typeId, id, err = p.ReadFieldBegin()

	if pw.isServer { //server side

		if id != SW_MAGIC_FIELD_ID || typeId != thrift.MAP {
			//fmt.Printf("[%s] ReadFieldBegin: %v, %p-%d\n", pw.operationName, routine.Goid(), pw, len(pw.header))
			return
		}
		_, _, size, ee := p.ReadMapBegin()
		if ee != nil || size == 0 {
			//fmt.Printf("[%s] begin: %v zzzzzzzzzzzzzzzzzzzzzzz %+v\n", pw.operationName, routine.Goid(), err)
			err = ee
			p.ReadMapEnd()
			p.ReadFieldEnd()
			name, typeId, id, err = pw.ReadFieldBegin()
			return
		}

		//fmt.Printf("[%s] readHeader begin: %v, %p-%d\n", pw.operationName, routine.Goid(), pw, len(pw.header))

		for i := 0; i < size; i++ {
			var key, value string
			var e1 error
			if key, e1 = p.ReadString(); e1 != nil {
				return
			}
			if value, e1 = p.ReadString(); e1 != nil {
				return
			}
			pw.header[key] = value
		}

		p.ReadMapEnd()
		p.ReadFieldEnd()
		name, typeId, id, err = pw.ReadFieldBegin()
	}
	return
}

func (pw *TProtocolWrapper) WriteMessageBegin(name string, typeId thrift.TMessageType, seqid int32) (err error) {
	pw.operationName = name

	if typeId == thrift.CALL { //client side.
		pw.isClient = true
		pw.headerInjected = false
		pw.header = nil
		//pw.createExitSpan()

		//entryCtx, span, entrySpan := recoverFromContext()
		//if entryCtx == nil {
		//	entryCtx = context.Background()
		//}
		//storeToContext(entryCtx, span, entrySpan)

	} else { //server side echo
		_, _, pw.entrySpan = recoverFromContext()
		if pw.entrySpan != nil {
			if typeId == thrift.EXCEPTION || typeId == thrift.INVALID_TMESSAGE_TYPE { //本地发生的错误
				//fmt.Printf("BBBBBBBB\n")
				pw.entrySpan.Error(time.Now(), fmt.Sprintf("%s error %d", name, typeId)) //错误上报，第二个参数开始（可选）表示错误日志。
				pw.entrySpan.Tag(go2sky.TagStatusCode, "1")                              //错误代码上报，可选。
			} else {
				//pw.entrySpan.Log(time.Now(), name+" suc")   //日志上报，可选。
				pw.entrySpan.Tag(go2sky.TagStatusCode, "0") //错误代码上报，可选。
			}
			pw.entrySpan.End()
		}

	}

	return pw.TProtocol.WriteMessageBegin(name, typeId, seqid)
}

func (pw *TProtocolWrapper) WriteMessageEnd() error {
	if pw.isClient {
		//fmt.Printf("WriteMessageEnd [%s] : %v, %p, %p-%d\n", pw.operationName, routine.Goid(), entryCtx, pw, len(pw.header))
		//fmt.Printf("WriteMessageEnd [%s] : %v, %p-%d\n", pw.operationName, routine.Goid(), pw, len(pw.header))

	}
	return pw.TProtocol.WriteMessageEnd()

}

func (pw *TProtocolWrapper) WriteFieldBegin(name string, typeId thrift.TType, id int16) error {
	if pw.isClient { //client side.
		if id == SW_MAGIC_FIELD_ID || typeId == thrift.MAP {
			fmt.Printf("[%s] WriteFieldBegin ignore: %v, %p-%d\n", pw.operationName, routine.Goid(), pw, len(pw.header))
			pw.ignoreField = true
			return nil
		}
	}
	return pw.TProtocol.WriteFieldBegin(name, typeId, id)
}

func (pw *TProtocolWrapper) WriteFieldEnd() error {
	if pw.isClient { //client side.
		if pw.ignoreField {
			pw.ignoreField = false
			return nil
		}
	}
	return pw.TProtocol.WriteFieldEnd()
}

func (pw *TProtocolWrapper) WriteMapBegin(keyType thrift.TType, valueType thrift.TType, size int) error {
	if pw.isClient { //client side.
		if pw.ignoreField {
			return nil
		}
	}
	return pw.TProtocol.WriteMapBegin(keyType, valueType, size)
}

func (pw *TProtocolWrapper) WriteMapEnd() error {
	if pw.isClient { //client side.
		if pw.ignoreField {
			return nil
		}
	}
	return pw.TProtocol.WriteMapEnd()
}

func (pw *TProtocolWrapper) WriteString(value string) error {
	if pw.isClient { //client side.
		if pw.ignoreField {
			return nil
		}
	}
	return pw.TProtocol.WriteString(value)
}

func (pw *TProtocolWrapper) WriteFieldStop() (err error) {
	if pw.isClient && !pw.headerInjected { //client side.
		pw.createExitSpan()
		//fmt.Printf("[%s] WriteFieldStop inject: %v, %p-%d\n", pw.operationName, routine.Goid(), pw, len(pw.header))

		pw.headerInjected = true
		pw.TProtocol.WriteFieldBegin(SW_MAGIC_FIELD, thrift.MAP, SW_MAGIC_FIELD_ID)
		pw.TProtocol.WriteMapBegin(thrift.STRING, thrift.STRING, len(pw.header))
		for key, value := range pw.header {
			pw.TProtocol.WriteString(key)
			pw.TProtocol.WriteString(value)
		}
		pw.TProtocol.WriteMapEnd()
		pw.TProtocol.WriteFieldEnd()

	} else {
		//fmt.Printf("[%s] WriteFieldStop2: %v, %p-%d\n", pw.operationName, routine.Goid(), pw, len(pw.header))
	}
	return pw.TProtocol.WriteFieldStop()
}

func (pw *TProtocolWrapper) createExitSpan() {

	//pw.spanCreated = true
	pw.header = make(map[string]string)

	var span, entrySpan go2sky.Span
	var entryCtx context.Context

	entryCtx, _, entrySpan = recoverFromContext()
	//fmt.Printf("WriteFieldStop createExitSpan [%s] : %v, %p, %p-%d\n", pw.operationName, routine.Goid(), entryCtx, pw, len(pw.header))

	var parentCtx context.Context
	if entryCtx == nil {
		//fmt.Println("entryCtx should not be nil")
		//entrySpan, entryCtx, err = pw.tracer.CreateLocalSpan(context.Background())
		parentCtx = context.Background()
	} else {
		parentCtx = entryCtx
	}

	var err error
	span, err = pw.tracer.CreateExitSpan(parentCtx, pw.operationName, pw.reqAddr, func(key, value string) error {
		pw.header[key] = value

		return nil
	})
	if err != nil {
		fmt.Println("CreateExitSpan error:", err)
		return
	}
	//defer span.End()
	pw.span = span
	storeToContext(entryCtx, span, entrySpan)
	//fmt.Printf("WriteFieldStop span [%s] : %p\n", pw.operationName, span)

	if span != nil {
		span.SetComponent(componentIDGOThriftClient)
		span.SetSpanLayer(v3.SpanLayer_RPCFramework)
		span.SetOperationName(pw.operationName)
	}
	return

}

//func (pw *TProtocolWrapper) Skip(fieldType thrift.TType) (err error) {
//	err = pw.TProtocol.Skip(fieldType)
//	return
//}
