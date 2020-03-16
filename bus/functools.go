package bus

import "fmt"

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

// A Func is a function with an argument which can return an error when it fails.
type Func func(interface{}) error

// A Function encapsulates a Func which can be called with an argument. The invocation will be delegated through
// nsq so multiple instances of the same function can run in different processes. Only one of them
// will be invoked.
type Function struct {
	endpoints *Endpoints
	cr        *ConsumerRegistration
	fn        Func
	name      string
}

type funcParam struct {
	Parameter interface{} `json:"parameter"`
}

// Function creates a Function from the with the given endpoints. The name of the function will be a
// distributed selector for the given go function. So every function which is registered with the same
// name can receive the invocation inside the cluster.
func (e *Endpoints) Function(name string, fn Func) (*Function, error) {
	reg, err := e.consumer.Register(name, "function#ephemeral")
	if err != nil {
		return nil, fmt.Errorf("cannot register consumer for function %q: %w", name, err)
	}
	cb := &Function{
		endpoints: e,
		cr:        reg,
		fn:        fn,
		name:      name,
	}
	if err = reg.Consume(funcParam{}, cb.receive, 5); err != nil {
		return nil, fmt.Errorf("cannot consume: %w", err)
	}
	return cb, nil
}

func (f *Function) receive(par interface{}) error {
	fp := par.(*funcParam)
	return f.fn(fp.Parameter)
}

// MustSucceed invokes the function with no limit. So nsq will invoke the connected go function
// until no error is returned.
func (f *Function) MustSucceed(arg interface{}) error {
	return f.endpoints.publisher.Publish(f.name, funcParam{Parameter: arg})
}
