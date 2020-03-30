package bus

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

// Endpoints couples a consumer and a publisher to a single entity.
type Endpoints struct {
	consumer  *Consumer
	publisher Publisher
}

// NewEndpoints creates the Endpoints for the given publisher and consumer.
func NewEndpoints(consumer *Consumer, publisher Publisher) *Endpoints {
	return &Endpoints{
		consumer:  consumer,
		publisher: publisher,
	}
}

// DirectEndpoints returns endpoints which call the target function directly. You do not need
// a running nsq for this. This can be used with unit tests. It should not be used in production
// code because the invocation of functions will not be persistent and will be delegated to the
// current running process by forking a gorouting to call the receiving function.
func DirectEndpoints() *Endpoints {
	return &Endpoints{}
}

// A Function encapsulates a Func which can be called with an argument. The invocation will be delegated through
// nsq so multiple instances of the same function can run in different processes. Only one of them
// will be invoked.
type Function struct {
	endpoints *Endpoints
	cr        *ConsumerRegistration
	fn        reflect.Value
	name      string
}

// Function creates a Function from the the given endpoints. The name of the function will be a
// distributed selector for the given go function. So every function which is registered with the same
// name can receive the invocation inside the cluster.
// The function must be a normal go function with one parameter and one result of type error:
//   ep := NewEndpoints(...)
//   f, err := ep.Function("hello", func (s string) error {
//      fmt.Printf("Hello %s\n", s)
//      return nil
//   })
//   f.Must("world"); // prints "Hello world"
// The target function can receive structs or pointer to structs. Please notice that when using
// `DirectEndpoints` the parameters are not marshalled/unmarshalled via JSON, so using addresses
// can have side effects.
func (e *Endpoints) Function(name string, fn interface{}) (*Function, error) {
	fntype := reflect.TypeOf(fn)
	if fntype.Kind() != reflect.Func {
		return nil, fmt.Errorf("the function parameter must be a function")
	}
	if fntype.NumIn() != 1 {
		return nil, fmt.Errorf("the number of parameters in the function must be one")
	}
	if fntype.NumOut() != 1 {
		return nil, fmt.Errorf("the function must return exactly one value of type error")
	}
	errtype := reflect.TypeOf(errors.New(""))
	if !errtype.AssignableTo(fntype.Out(0)) {
		return nil, fmt.Errorf("the return type is not of type 'error'")
	}

	if e.consumer == nil || e.publisher == nil {
		// someone wants a local function
		return &Function{name: name, fn: reflect.ValueOf(fn)}, nil
	}
	if err := e.publisher.CreateTopic(name); err != nil {
		return nil, fmt.Errorf("cannot create topic: %q: %w", name, err)
	}
	reg, err := e.consumer.Register(name, "function")
	if err != nil {
		return nil, fmt.Errorf("cannot register consumer for function %q: %w", name, err)
	}
	cb := &Function{
		endpoints: e,
		cr:        reg,
		fn:        reflect.ValueOf(fn),
		name:      name,
	}
	partype := fntype.In(0)
	for partype.Kind() == reflect.Ptr {
		partype = partype.Elem()
	}
	pvalue := reflect.New(partype).Elem()
	if err = reg.Consume(pvalue.Interface(), cb.receive, 5); err != nil {
		return nil, fmt.Errorf("cannot consume: %w", err)
	}
	return cb, nil
}

// receive will be called when the target function has to be invoked. we check
// here if the given value and the target parameter type "match" in a form
// that the caller can mix value and pointer types. If the target function
// receives a value type, a value will be passed to it. If it needs a pointer
// a pointer will be passed if there is one; if the function is invoked with a
// value type, this value will be copied so we can pass a pointer to the target
// function.
func (f *Function) receive(par interface{}) error {
	v := reflect.ValueOf(par)
	vkind := reflect.TypeOf(par).Kind()
	pkind := f.fn.Type().In(0).Kind()

	parms := []reflect.Value{v}
	if vkind != pkind {
		if pkind == reflect.Ptr {
			// function wants a ptr but we got a value
			// --> copy value and pass pointer to this copy
			nv := reflect.New(reflect.TypeOf(par))
			nv.Elem().Set(v)
			parms = []reflect.Value{nv}
		} else if vkind == reflect.Ptr {
			// function wants value
			parms = []reflect.Value{v.Elem()}
		}
	}
	res := f.fn.Call(parms)
	if res[0].IsNil() {
		return nil
	}
	return res[0].Interface().(error)
}

// Must invokes the function with no limit. So nsq will invoke the connected go function
// until no error is returned. The function itself returns an error if there is a
// communication problem with nsq.
func (f *Function) Must(arg interface{}) error {
	if f.endpoints == nil {
		go func(arg interface{}) {
			// local function. this is not the "normal" use case so here we do a
			// simple fork of a goroutine. it is up to the target function to
			// return a nil value. if no nil value is returned ever, this goroutine
			// will never end!
			for {
				if err := f.receive(arg); err == nil {
					return
				}
				time.Sleep(time.Millisecond * 100)
			}
		}(arg)
		return nil
	}
	return f.endpoints.publisher.Publish(f.name, arg)
}
