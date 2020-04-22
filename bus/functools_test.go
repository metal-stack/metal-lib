package bus

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"testing"
)

func TestFunctionWithWrongParams(t *testing.T) {
	e := NewEndpoints(consumer, publisher)
	_, _, err := e.Function("helloworld", func(arg1, arg2 string) error {
		return nil
	})
	if err == nil {
		t.Errorf("function creation should fail: must only have one parameter")
	}
}

func TestFunctionWithWrongResults(t *testing.T) {
	e := NewEndpoints(consumer, publisher)
	_, _, err := e.Function("helloworld", func(arg1 string) (string, error) {
		return "", nil
	})
	if err == nil {
		t.Errorf("function creation should fail: must only return one result")
	}
}

func TestFunctionWithWrongResult(t *testing.T) {
	e := NewEndpoints(consumer, publisher)
	_, _, err := e.Function("helloworld", func(arg1 string) string {
		return ""
	})
	if err == nil {
		t.Errorf("function creation should fail: must return an error")
	}
}

func TestFunctionWithWrongFunc(t *testing.T) {
	e := NewEndpoints(consumer, publisher)
	_, _, err := e.Function("helloworld", struct{}{})
	if err == nil {
		t.Errorf("function creation should fail: must get a function as parameter")
	}
}

func TestFunctionHelloWorld(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	e := NewEndpoints(consumer, publisher)
	_, f, err := e.Function("helloworld", func(arg string) error {
		res = arg
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	value := "Hello world"
	err = f(value)
	wg.Wait()

	if err != nil {
		t.Fatalf("function must succeed, %v", err)
	}

	if res != value {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}

func TestUniqueFunctionWithoutFunc(t *testing.T) {
	ep := DirectEndpoints()
	_, _, _, err := ep.Unique("blubber", nil)
	if err == nil {
		t.Errorf("a unique function needs also a go func")
	}
}

func TestUniqueFunctionHelloWorld(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	e := NewEndpoints(consumer, publisher)
	_, f, _, err := e.Unique("uniquehelloworld", func(arg string) error {
		res = arg
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	value := "Hello world"
	err = f(value)
	wg.Wait()

	if err != nil {
		t.Fatalf("function must succeed, %v", err)
	}

	if res != value {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}

func TestTwoProcessesFunctionHelloWorld(t *testing.T) {
	value := "Hello world"
	e := NewEndpoints(consumer, publisher)
	if _, pub := os.LookupEnv("PUBLISH"); pub {
		// this unit-test was forked in another process. here we only publish a value
		_, f, err := e.Function("distributed-hello", nil)
		if err != nil {
			t.Errorf("cannot create function, %v", err)
		}
		err = f(value)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	go func() {
		// fork this unit-test in another process and set two env-variables, so the
		// other process can publish a value but does not start nsqd again
		executable, _ := os.Executable()
		args := []string{"-test.timeout=10s", "-test.run=^(TestTwoProcessesFunctionHelloWorld)$"}
		cmd := exec.Command(executable, args...)
		cmd.Env = append([]string{"PUBLISH=1", "NO_NSQD_START=1"}, os.Environ()...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("error occured: %s", string(out))
		}
	}()
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	_, _, err := e.Function("distributed-hello", func(arg string) error {
		res = arg
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	wg.Wait()

	if res != value {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}

func TestUniqueTargetFunctionWithResponse(t *testing.T) {
	// create two functions, one with a unique name and send this name to the well-known
	// service. the service then calls this unique-function name with the response
	value := "Hello world"
	e := NewEndpoints(consumer, publisher)
	if _, pub := os.LookupEnv("PRODUCER"); pub {
		// this unit-test was forked in another process. here we only publish a value
		var wg sync.WaitGroup
		wg.Add(1)
		_, _, err := e.Function("hello-service", func(arg string) error {
			_, result, err := e.Client(arg)
			if err != nil {
				log.Fatal(err)
			}
			err = result(value)
			wg.Done()
			return err
		})
		if err != nil {
			t.Errorf("cannot create function, %v", err)
		}
		wg.Wait()
		return
	}
	go func() {
		// fork this unit-test in another process and set two env-variables, so the
		// other process can publish a value but does not start nsqd again
		executable, _ := os.Executable()
		args := []string{"-test.timeout=10s", "-test.run=^(TestUniqueTargetFunctionWithResponse)$"}
		cmd := exec.Command(executable, args...)
		cmd.Env = append([]string{"PRODUCER=1", "NO_NSQD_START=1"}, os.Environ()...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("error occured: %s", string(out))
		}
	}()
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	_, _, cbname, err := e.Unique("hello-result", func(arg string) error {
		res = arg
		wg.Done()
		return nil
	})
	if err != nil {
		t.Fatalf("cannot create function, %v", err)
	}
	_, srv, err := e.Function("hello-service", nil)
	if err != nil {
		t.Fatalf("cannot create function, %v", err)
	}
	if err = srv(cbname); err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if res != value {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}

func TestFunctionRetry(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	num := 0
	retries := 1
	e := NewEndpoints(consumer, publisher)
	_, f, err := e.Function("helloworld-retry", func(arg string) error {
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
	err = f(value)
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
	_, f, err := e.Function("hellostruct", func(arg *testStruct) error {
		res = arg.Name
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	value := "Hello world"
	err = f(testStruct{Name: value})
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
	_, f, err := e.Function("direct-hellostruct", func(arg *testStruct) error {
		res = arg.Name
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	value := "Hello world"
	err = f(&testStruct{Name: value})
	wg.Wait()

	if err != nil {
		t.Fatalf("function must succeed, %v", err)
	}

	if res != value {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}

func TestDirectFunctionWithDifferentParameters(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	e := DirectEndpoints()
	_, f, err := e.Function("direct-hellostructpointer", func(arg *testStruct) error {
		res = arg.Name
		wg.Done()
		return nil
	})
	if err != nil {
		t.Errorf("cannot create function, %v", err)
	}

	value := "Hello world"
	err = f(testStruct{Name: value})
	wg.Wait()

	if err != nil {
		t.Fatalf("function must succeed, %v", err)
	}

	if res != value {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}

func TestDirectFunctionRetry(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	res := ""
	num := 0
	retries := 1
	e := DirectEndpoints()
	_, f, err := e.Function("helloworld-retry-direct", func(arg string) error {
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
	err = f(value)
	wg.Wait()

	if err != nil {
		t.Fatalf("function must succeed, %v", err)
	}

	if res != fmt.Sprintf("%s: %d", value, retries) {
		t.Errorf("result is %q, but should be %q", res, value)
	}
}
