package server

import (
	"fmt"
	"github.com/ankur-anand/simple-go-rpc/src/dataserial"
	"github.com/ankur-anand/simple-go-rpc/src/transport"
	"io"
	"log"
	"net"
	"reflect"
)

// RPCServer ...
type RPCServer struct {
	addr  string
	funcs map[string]reflect.Value
}

// NewServer creates a new server
func NewServer(addr string) *RPCServer {
	return &RPCServer{addr: addr, funcs: make(map[string]reflect.Value)}
}

// Register the name of the function and its entries
func (s *RPCServer) Register(fnName string, fFunc interface{}) {
	if _, ok := s.funcs[fnName]; ok {
		return
	}

	s.funcs[fnName] = reflect.ValueOf(fFunc)
}

// Execute the given function if present
func (s *RPCServer) Execute(req dataserial.RPCdata) dataserial.RPCdata {
	// get method by name
	f, ok := s.funcs[req.Name]
	if !ok {
		// since method is not present
		e := fmt.Sprintf("func %s not Registered", req.Name)
		log.Println(e)
		return dataserial.RPCdata{Name: req.Name, Args: nil, Err: e}
	}

	log.Printf("func %s is called\n", req.Name)
	// unpack request arguments
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
	return dataserial.RPCdata{Name: req.Name, Args: resArgs, Err: er}
}

// Run server
func (s *RPCServer) Run() {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Printf("listen on %s err: %v\n", s.addr, err)
		return
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept err: %v\n", err)
			continue
		}
		go func() {
			connTransport := transport.NewTransport(conn)
			for {
				// read request
				req, err := connTransport.Read()
				if err != nil {
					if err != io.EOF {
						log.Printf("read err: %v\n", err)
						return
					}
				}

				// decode the data and pass it to execute
				decReq, err := dataserial.Decode(req)
				if err != nil {
					log.Printf("Error Decoding the Payload err: %v\n", err)
					return
				}
				// get the executed result.
				resP := s.Execute(decReq)
				// encode the data back
				b, err := dataserial.Encode(resP)
				if err != nil {
					log.Printf("Error Encoding the Payload for response err: %v\n", err)
					return
				}
				// send response to client
				err = connTransport.Send(b)
				if err != nil {
					log.Printf("transport write err: %v\n", err)
				}
			}
		}()
	}
}
