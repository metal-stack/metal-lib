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

// DirectEndpoints returns endpoints which call the target function directly.
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

// Function creates a Function from the with the given endpoints. The name of the function will be a
// distributed selector for the given go function. So every function which is registered with the same
// name can receive the invocation inside the cluster.
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
		// someone wants a sync function
		return &Function{name: name, fn: reflect.ValueOf(fn)}, nil
	}
	if err := e.publisher.CreateTopic(name); err != nil {
		return nil, fmt.Errorf("cannot create topic: %q: %w", name, err)
	}
	reg, err := e.consumer.Register(name, "function#ephemeral")
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

func (f *Function) receive(par interface{}) error {
	v := reflect.ValueOf(par)
	vkind := reflect.TypeOf(par).Kind()
	pkind := f.fn.Type().In(0).Kind()

	parms := []reflect.Value{v}
	if vkind != pkind {
		if pkind == reflect.Ptr {
			// function wants a ptr
			parms = []reflect.Value{v.Addr()}
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
		//sync function
		for {
			if err := f.receive(arg); err == nil {
				return nil
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
	return f.endpoints.publisher.Publish(f.name, arg)
}
