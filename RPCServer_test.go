package main

import (
	"testing"
	"reflect"
)

// rpc function for test purpose
func add(a, b int) (int, error) {
	return a + b, nil
}

func TestRPCServer_ExecuteAndRegister(t *testing.T) {
	funcs := make(map[string]reflect.Value)
	s := &RPCServer{funcs: funcs}
	s.Register("add", add)
	dataSlice := []int{1, 2}
	interfaceSlice := make([]interface{}, len(dataSlice))
	for i, d := range dataSlice {
		interfaceSlice[i] = d
	}
	// make a request.
	reqRPC := RPCdata{
		Name: "add",                                                   // name of the function
		Args: interfaceSlice, // request's or response's body expect error.
		Err:  "",                                                      // Not Present for request
	}
	res := s.Execute(reqRPC)
	if res.Name != "add" && res.Err != "" {
		t.Fatalf("Expected Response Function Name to be add Got %s or res.Err is not null %s", res.Name, res.Err)
	}
	v, ok := res.Args[0].(int)
	if !ok {
		t.Fatalf("Expected Return value of type int")
	}
	if len(res.Args) != 1 && v != 3 {
		t.Fatalf("Expected Argument response Argument of type int with value 3 and len 1")
	}
}
