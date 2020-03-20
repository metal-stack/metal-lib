package bus

import (
	"fmt"
	"sync"
	"testing"
)

func TestFunctionWithWrongParams(t *testing.T) {
	e := NewEndpoints(consumer, publisher)
	_, err := e.Function("helloworld", func(arg1, arg2 string) error {
		return nil
	})
	if err == nil {
		t.Errorf("function creation should fail: must only have one parameter")
	}
}

func TestFunctionWithWrongResults(t *testing.T) {
	e := NewEndpoints(consumer, publisher)
	_, err := e.Function("helloworld", func(arg1 string) (string, error) {
		return "", nil
	})
	if err == nil {
		t.Errorf("function creation should fail: must only return one result")
	}
}

func TestFunctionWithWrongResult(t *testing.T) {
	e := NewEndpoints(consumer, publisher)
	_, err := e.Function("helloworld", func(arg1 string) string {
		return ""
	})
	if err == nil {
		t.Errorf("function creation should fail: must return an error")
	}
}

func TestFunctionWithWrongFunc(t *testing.T) {
	e := NewEndpoints(consumer, publisher)
	_, err := e.Function("helloworld", struct{}{})
	if err == nil {
		t.Errorf("function creation should fail: must get a function as parameter")
	}
}

func TestFunctionHelloWorld(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	e := NewEndpoints(consumer, publisher)
	f, err := e.Function("helloworld", func(arg string) error {
		res = arg
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	value := "Hello world"
	err = f.Must(value)
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
	f, err := e.Function("helloworld-replay", func(arg string) error {
		if num < retries {
			num += 1
			return fmt.Errorf("not on the first run: %d", num)
		}
		res = fmt.Sprintf("%s: %d", arg, num)
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	value := "Hello world"
	err = f.Must(value)
	wg.Wait()

	if err != nil {
		t.Fatalf("function must succeed, %v", err)
	}

	if res != fmt.Sprintf("%s: %d", value, retries) {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}

type testStruct struct {
	Name string
}

func TestFunctionWithStruct(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	e := NewEndpoints(consumer, publisher)
	f, err := e.Function("hellostruct", func(arg *testStruct) error {
		res = arg.Name
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	value := "Hello world"
	err = f.Must(testStruct{Name: value})
	wg.Wait()

	if err != nil {
		t.Fatalf("function must succeed, %v", err)
	}

	if res != value {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}

func TestDirectFunctionWithStruct(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	e := DirectEndpoints()
	f, err := e.Function("direct-hellostruct", func(arg *testStruct) error {
		res = arg.Name
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	value := "Hello world"
	err = f.Must(&testStruct{Name: value})
	wg.Wait()

	if err != nil {
		t.Fatalf("function must succeed, %v", err)
	}

	if res != value {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}
