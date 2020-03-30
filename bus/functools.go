package bus

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

// Endpoints couples a consumer and a publisher to a single entity.
type Endpoints struct {
	consumer  *Consumer
	publisher Publisher
}

// NewEndpoints creates the Endpoints for the given publisher and consumer. If one of the values
// is nil, the function created by this endpoint can only be used for invoking or compute but not both.
// If both values are set, the function will be invoked by the same process which implements it otherwise
// you have some sort of client and server for the function.
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
	fn        reflect.Value
	name      string
}

type Func func(interface{}) error

// Function creates a Function from the the given endpoints. The name of the function will be a
// distributed selector for the given go function. So every function which is registered with the same
// name can receive the invocation inside the cluster.
// The function must be a normal go function with one parameter and one result of type error:
//   ep := NewEndpoints(...)
//   f, err := ep.Function("hello", func (s string) error {
//      fmt.Printf("Hello %s\n", s)
//      return nil
//   })
//   f("world"); // prints "Hello world"
// The target function can receive structs or pointer to structs. Please notice that when using
// `DirectEndpoints` the parameters are not marshalled/unmarshalled via JSON, so using addresses
// can have side effects.
func (e *Endpoints) Function(name string, fn interface{}) (Func, error) {
	return e.function(name, "function", fn, true)
}

// Unique usese an unique, ephemeral topic so the topic will be deregisted when there is no
// consumer any more for this function. Use this function to create a unique receiver, so function
// invocations will not be distributed and the topic only exists as long as the registration
// process is active. The name of this unique function is returned so it can be used with the
// `Target` function to invoke it.
func (e *Endpoints) Unique(name string, fn interface{}) (Func, string, error) {
	id := uuid.New().String()
	topic := name + "-" + id + "#ephemeral"
	f, err := e.function(topic, "function", fn, true)
	return f, topic, err
}

func (e *Endpoints) function(name, chanName string, fn interface{}, dopublish bool) (Func, error) {
	if fn != nil {
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
	}
	if e.consumer == nil && e.publisher == nil {
		// someone wants a local function
		f := &Function{name: name, fn: reflect.ValueOf(fn)}
		return f.invoker(), nil
	}
	if e.publisher != nil {
		if err := e.publisher.CreateTopic(name); err != nil {
			return nil, fmt.Errorf("cannot create topic: %q: %w", name, err)
		}
	}
	cb := &Function{
		endpoints: e,
		fn:        reflect.ValueOf(fn),
		name:      name,
	}
	if e.consumer != nil && fn != nil {
		reg, err := e.consumer.Register(name, chanName)
		if err != nil {
			return nil, fmt.Errorf("cannot register consumer for function %q: %w", name, err)
		}
		partype := reflect.TypeOf(fn).In(0)
		for partype.Kind() == reflect.Ptr {
			partype = partype.Elem()
		}
		pvalue := reflect.New(partype).Elem()
		if err = reg.Consume(pvalue.Interface(), cb.receive, 5); err != nil {
			return nil, fmt.Errorf("cannot consume: %w", err)
		}
	}
	return cb.invoker(), nil
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
	pkind := vkind
	if !f.fn.IsZero() {
		pkind = f.fn.Type().In(0).Kind()
	}

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

func (f *Function) invoker() Func {
	return func(arg interface{}) error {
		return f.must(arg)
	}
}

// must invokes the function with no limit. So nsq will invoke the connected go function
// until no error is returned. The function itself returns an error if there is a
// communication problem with nsq.
func (f *Function) must(arg interface{}) error {
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
