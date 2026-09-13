package defaults_test

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/creasty/defaults"
)

// Config implements defaults.Setter for ExampleSetter below.
type Config struct {
	Retries int           `default:"3"`
	Backoff time.Duration `default:"-"`
}

// SetDefaults derives a default that a tag cannot express, and CanUpdate keeps it from overriding
// what the caller already provided.
func (c *Config) SetDefaults() {
	if defaults.CanUpdate(c.Backoff) {
		c.Backoff = time.Duration(c.Retries) * time.Second
	}
}

// Endpoint implements defaults.Setter for Example below.
type Endpoint struct {
	Host string `default:"localhost"`
	Port int    `default:"8080"`
	Addr string `default:"-"`
}

// SetDefaults derives Addr from Host and Port, which the tags have filled by the time it is called.
func (e *Endpoint) SetDefaults() {
	if defaults.CanUpdate(e.Addr) {
		e.Addr = net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
	}
}

// This example fills a tree of structs, pointers, slices and maps in one call. Every Endpoint the
// call reaches gets the same treatment wherever it sits: its tags, then its SetDefaults.
func Example() {
	type Gateway struct {
		// A struct is filled without needing a tag.
		Listen Endpoint
		// A nil pointer needs a tag to be allocated. The tag's JSON goes in first, and the
		// struct's own tags fill what it leaves out.
		Metrics *Endpoint `default:"{\"Port\": 9090}"`
		// Each element a tag creates is filled too.
		Backends []Endpoint `default:"[{\"Host\": \"app-1\"}, {\"Host\": \"app-2\"}]"`
		// So is each element the caller supplied, keeping the values it already holds.
		Routes map[string]Endpoint
		// An opted-out struct is not descended into, so its SetDefaults is not called either.
		Tracing Endpoint `default:"-"`
	}

	gateway := Gateway{
		Routes: map[string]Endpoint{"/api": {Port: 3000}},
	}
	defaults.MustSet(&gateway)

	fmt.Println(gateway.Listen.Addr)
	fmt.Println(gateway.Metrics.Addr)
	fmt.Println(gateway.Backends[0].Addr, gateway.Backends[1].Addr)
	fmt.Println(gateway.Routes["/api"].Addr)
	fmt.Println(gateway.Tracing == Endpoint{})
	// Output:
	// localhost:8080
	// localhost:9090
	// app-1:8080 app-2:8080
	// localhost:3000
	// true
}

func ExampleSet() {
	type Server struct {
		Host    string            `default:"localhost"`
		Port    int               `default:"8080"`
		Timeout time.Duration     `default:"30s"`
		Tags    []string          `default:"[\"web\"]"`
		Labels  map[string]string `default:"{\"env\": \"dev\"}"`
	}

	var server Server
	if err := defaults.Set(&server); err != nil {
		panic(err)
	}

	fmt.Printf("%s:%d\n", server.Host, server.Port)
	fmt.Println(server.Timeout)
	fmt.Println(server.Tags, server.Labels)
	// Output:
	// localhost:8080
	// 30s
	// [web] map[env:dev]
}

func ExampleSet_nested() {
	type Retry struct {
		Attempts int `default:"3"`
	}
	type Client struct {
		// A struct is descended into with or without a tag, and a pointer needs one to be
		// allocated at all.
		Retry    Retry
		Fallback *Retry `default:"{}"`
		Disabled *Retry
	}

	var client Client
	if err := defaults.Set(&client); err != nil {
		panic(err)
	}

	fmt.Println(client.Retry.Attempts)
	fmt.Println(client.Fallback.Attempts)
	fmt.Println(client.Disabled == nil)
	// Output:
	// 3
	// 3
	// true
}

func ExampleSetter() {
	var config Config
	if err := defaults.Set(&config); err != nil {
		panic(err)
	}
	fmt.Println(config.Retries, config.Backoff)

	// A value the caller supplied is left alone.
	custom := Config{Backoff: time.Minute}
	if err := defaults.Set(&custom); err != nil {
		panic(err)
	}
	fmt.Println(custom.Retries, custom.Backoff)
	// Output:
	// 3 3s
	// 3 1m0s
}

func ExampleCanUpdate() {
	fmt.Println(defaults.CanUpdate(0), defaults.CanUpdate(123))
	fmt.Println(defaults.CanUpdate(""), defaults.CanUpdate("hello"))
	// Output:
	// true false
	// true false
}
