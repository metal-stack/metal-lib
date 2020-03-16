package bus

import (
	"fmt"
	"sync"
	"testing"
)

func TestFunctionHelloWorld(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	e := NewEndpoints(consumer, publisher)
	f, err := e.Function("helloworld", func(arg interface{}) error {
		res = fmt.Sprintf("%v", arg)
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	value := "Hello world"
	err = f.MustSucceed(value)
	wg.Wait()

	if err != nil {
		t.Fatalf("function must succeed, %v", err)
	}

	if res != value {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}

func TestFunctionReplay(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	num := 0
	retries := 1
	e := NewEndpoints(consumer, publisher)
	f, err := e.Function("helloworld-replay", func(arg interface{}) error {
		if num < retries {
			num += 1
			return fmt.Errorf("not on the first run: %d", num)
		}
		res = fmt.Sprintf("%v: %d", arg, num)
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	value := "Hello world"
	err = f.MustSucceed(value)
	wg.Wait()

	if err != nil {
		t.Fatalf("function must succeed, %v", err)
	}

	if res != fmt.Sprintf("%s: %d", value, retries) {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}
