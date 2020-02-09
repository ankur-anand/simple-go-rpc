## Simple GoRPC

Learning RPC basic building blocks by building a simple RPC framework in Golang from scratch.

## RPC

In Simple Term **Service A** wants to call **Service B** functions. But those two services are not in the same memory space. So it cannot be called directly.

So, in order to make this call happen, we need to express the semantics of how to call and also how to pass the communication through the network.

### Let's think what we do when we call function in the same memory space (local call)

```go
type User struct {
	Name string
	Age int
}

var userDB = map[int]User{
	1: User{"Ankur", 85},
	9: User{"Anand", 25},
	8: User{"Ankur Anand", 27},
}


func QueryUser(id int) (User, error) {
	if u, ok := userDB[id]; ok {
		return u, nil
	}

	return User{}, fmt.Errorf("id %d not in user db", id)
}


func main() {
	u , err := QueryUser(8)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("name: %s, age: %d \n", u.Name, u.Age)
}
```

Now, how do we do the same function call over the network?

**Client** will call _QueryUser(id int)_ function over the network and there will be one server which will Serve the Call to this function and return the Response _User{"Name", id}, nil_.

## Network Transmission Data format.

Simple-gorpc will do TLV (fixed-length header + variable-length message body) encoding scheme to regulate the transmission of data, over the tcp.
**More on this later**

### Before we send our data over the network we need to define the structure how we are going to send the data over the network.

This helps us to define a common protocol that, the client and server both can understand. (protobuf IDL define what both server and client understand).

### So data received by the server needs to have:

- the name of the function to be called
- list of parameters to be passed to that function

Also let's agree that the second return value is of type error, indicating the RPC call result.

```go
// RPCdata transmission format
type RPCdata struct {
	Name string        // name of the function
	Args []interface{} // request's or response's body expect error.
	Err  string        // Error any executing remote server
}
```

So now that we have a format, we need to serialize this so that we can send it over the network.
In our case we will use the `go` default binary serialization protocol for encoding and decoding.

```go
// be sent over the network.
func Encode(data RPCdata) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode the binary data into the Go struct
func Decode(b []byte) (RPCdata, error) {
	buf := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buf)
	var data RPCdata
	if err := decoder.Decode(&data); err != nil {
		return Data{}, err
	}
	return data, nil
}
```

### Network Transmission

The reason for choosing the TLV protocol is due to the fact that it's very simple to implement, and it also fullfills our need over identification of the length of data to read, as we need to identify the number of bytes to read for this request over the stream of incoming request. `Send and Receive does the same`

```go
// Transport will use TLV protocol
type Transport struct {
	conn net.Conn // Conn is a generic stream-oriented network connection.
}

// NewTransport creates a Transport
func NewTransport(conn net.Conn) *Transport {
	return &Transport{conn}
}

// Send TLV data over the network
func (t *Transport) Send(data []byte) error {
	// we will need 4 more byte then the len of data
	// as TLV header is 4bytes and in this header
	// we will encode how much byte of data
	// we are sending for this request.
	buf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(data)))
	copy(buf[4:], data)
	_, err := t.conn.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

// Read TLV sent over the wire
func (t *Transport) Read() ([]byte, error) {
	header := make([]byte, 4)
	_, err := io.ReadFull(t.conn, header)
	if err != nil {
		return nil, err
	}
	dataLen := binary.BigEndian.Uint32(header)
	data := make([]byte, dataLen)
	_, err = io.ReadFull(t.conn, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
```

Now that we have the DataFormat and Transport protocol defined. We need and **RPC Server** and **RPC CLient**

## RPC SERVER

RPC Server will receive the `RPCData` which will have an function Name.
**So we need to maintain and map that contains an function name to actual function mapping**

```go
// RPCServer ...
type RPCServer struct {
	addr string
	funcs map[string] reflect.Value
}

// Register the name of the function and its entries
func (s *RPCServer) Register(fnName string, fFunc interface{}) {
	if _,ok := s.funcs[fnName]; ok {
		return
	}

	s.funcs[fnName] = reflect.ValueOf(fFunc)
}
```

Now that we have the func registered, when we receive the request we will check if the name of func passed during the execution of the function is present or not. and then will execute it accordingly

```go
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
	resArgs := make([]interface{}, len(out) - 1)
	for i := 0; i < len(out) - 1; i ++ {
		// Interface returns the constant value stored in v as an interface{}.
		resArgs[i] = out[i].Interface()
	}

	// pack error argument
	var er string
	if e, ok := out[len(out) - 1].Interface().(error); ok {
		// convert the error into error string value
		er = e.Error()
	}
	return RPCdata{Name: req.Name, Args: resArgs, Err: er}
}
```

## RPC CLIENT

Since the concrete implementation of the function is on the server side, the client only has the prototype of the function, so we need complete prototype of the calling function, so that we can call it.

```go
func (c *Client) callRPC(rpcName string, fPtr interface{}) {
	container := reflect.ValueOf(fPtr).Elem()
	f := func(req []reflect.Value) []reflect.Value {
		cReqTransport := NewTransport(c.conn)
		errorHandler := func(err error) []reflect.Value {
			outArgs := make([]reflect.Value, container.Type().NumOut())
			for i := 0; i < len(outArgs)-1; i++ {
				outArgs[i] = reflect.Zero(container.Type().Out(i))
			}
			outArgs[len(outArgs)-1] = reflect.ValueOf(&err).Elem()
			return outArgs
		}

		// Process input parameters
		inArgs := make([]interface{}, 0, len(req))
		for _, arg := range req {
			inArgs = append(inArgs, arg.Interface())
		}

		// ReqRPC
		reqRPC := RPCdata{Name: rpcName, Args: inArgs}
		b, err := Encode(reqRPC)
		if err != nil {
			panic(err)
		}
		err = cReqTransport.Send(b)
		if err != nil {
			return errorHandler(err)
		}
		// receive response from server
		rsp, err := cReqTransport.Read()
		if err != nil { // local network error or decode error
			return errorHandler(err)
		}
		rspDecode, _ := Decode(rsp)
		if rspDecode.Err != "" { // remote server error
			return errorHandler(errors.New(rspDecode.Err))
		}

		if len(rspDecode.Args) == 0 {
			rspDecode.Args = make([]interface{}, container.Type().NumOut())
		}
		// unpackage response arguments
		numOut := container.Type().NumOut()
		outArgs := make([]reflect.Value, numOut)
		for i := 0; i < numOut; i++ {
			if i != numOut-1 { // unpackage arguments (except error)
				if rspDecode.Args[i] == nil { // if argument is nil (gob will ignore "Zero" in transmission), set "Zero" value
					outArgs[i] = reflect.Zero(container.Type().Out(i))
				} else {
					outArgs[i] = reflect.ValueOf(rspDecode.Args[i])
				}
			} else { // unpackage error argument
				outArgs[i] = reflect.Zero(container.Type().Out(i))
			}
		}

		return outArgs
	}
	container.Set(reflect.MakeFunc(container.Type(), f))
}
```

### Testing our framework

```go
package main

import (
	"encoding/gob"
	"fmt"
	"net"
)

type User struct {
	Name string
	Age  int
}

var userDB = map[int]User{
	1: User{"Ankur", 85},
	9: User{"Anand", 25},
	8: User{"Ankur Anand", 27},
}

func QueryUser(id int) (User, error) {
	if u, ok := userDB[id]; ok {
		return u, nil
	}

	return User{}, fmt.Errorf("id %d not in user db", id)
}

func main() {
	// new Type needs to be registered
	gob.Register(User{})
	addr := "localhost:3212"
	srv := NewServer(addr)

	// start server
	srv.Register("QueryUser", QueryUser)
	go srv.Run()

	// wait for server to start.
	time.Sleep(1 * time.Second)

	// start client
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}
	cli := NewClient(conn)

	var Query func(int) (User, error)
	cli.callRPC("QueryUser", &Query)

	u, err := Query(1)
	if err != nil {
		panic(err)
	}
	fmt.Println(u)

	u2, err := Query(8)
	if err != nil {
		panic(err)
	}
	fmt.Println(u2)
}
```

Output

```
2019/07/23 20:26:18 func QueryUser is called
{Ankur 85}
2019/07/23 20:26:18 func QueryUser is called
{Ankur Anand 27}
```

`go run main.go`
