package defaults_test

import (
	"fmt"
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
