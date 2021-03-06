package indexeddb

import (
	"runtime"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/johnstarich/go-wasm/log"
	"github.com/pkg/errors"
)

func processRequest(request js.Value) promise.GoPromise {
	resolve, reject, prom := promise.NewGoPromise()

	done := false
	var errFunc, successFunc js.Func
	errFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err := request.Get("error")
		log.PrintJSValues("Txn request failed:", err)
		go reject(err)
		errFunc.Release()
		successFunc.Release()
		done = true
		return nil
	})
	successFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result := request.Get("result")
		log.PrintJSValues("Txn request succeeded:", result)
		go resolve(result)
		errFunc.Release()
		successFunc.Release()
		done = true
		return nil
	})
	request.Call("addEventListener", "error", errFunc)
	request.Call("addEventListener", "success", successFunc)
	if txn := request.Get("transaction"); txn.Type() != js.TypeNull {
		go func() {
			for i := 0; i < 5; i++ {
				runtime.Gosched()
				log.PrintJSValues("sleeping, err:", txn.Get("error"))
				time.Sleep(1 * time.Second)
			}
			if !done {
				go reject(nil)
			}
		}()
	}
	return prom
}

func commitTxn(txn js.Value) (err error) {
	defer catch(&err)
	if txn.Type() != js.TypeNull {
		txn.Call("commit")
	}
	return nil
}

func await(prom promise.GoPromise) (js.Value, error) {
	log.Print("Awaiting Promise: ", prom)
	runtime.Gosched()
	val, errVal := promise.AwaitGo(prom)
	if errVal != nil {
		return js.Value{}, js.Error{Value: errVal.(js.Value)}
	}
	return val.(js.Value), nil
}

func catch(err *error) {
	r := recover()
	if r == nil {
		return
	}
	switch val := r.(type) {
	case error:
		*err = val
	case js.Value:
		*err = js.Error{Value: val}
	default:
		*err = errors.Errorf("%+v", val)
	}
}
