package main

import (
	"log"
	"reflect"
	"fmt"
)

// RPCServer ...
type RPCServer struct {
	addr  string
	funcs map[string]reflect.Value
}

// Register the name of the function and its entries
func (s *RPCServer) Register(fnName string, fFunc interface{}) {
	if _, ok := s.funcs[fnName]; ok {
		return
	}

	s.funcs[fnName] = reflect.ValueOf(fFunc)
}

// Execute the given function if present
func (s *RPCServer) Execute(req RPCdata) RPCdata {
	// get method by name
	f, ok := s.funcs[req.Name]
	if !ok {
		// since method is not present
		e := fmt.Sprintf("func %s not Registered", req.Name)
		log.Println(e)
		return RPCdata{Name: req.Name, Args: nil, Err: e}
	}

	log.Printf("func %s is called\n", req.Name)
	// unpackage request arguments
	inArgs := make([]reflect.Value, len(req.Args))
	for i := range req.Args {
		inArgs[i] = reflect.ValueOf(req.Args[i])
	}

	// invoke requested method
	out := f.Call(inArgs)
	// now since we have followed the function signature style where last argument will be an error
	// so we will pack the response arguments expect error.
	resArgs := make([]interface{}, len(out)-1)
	for i := 0; i < len(out)-1; i++ {
		// Interface returns the constant value stored in v as an interface{}.
		resArgs[i] = out[i].Interface()
	}

	// pack error argument
	var er string
	if _, ok := out[len(out)-1].Interface().(error); ok {
		// convert the error into error string value
		er = out[len(out)-1].Interface().(error).Error()
	}
	return RPCdata{Name: req.Name, Args: resArgs, Err: er}
}
