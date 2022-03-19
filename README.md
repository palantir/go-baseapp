# go-baseapp [![GoDoc](https://godoc.org/github.com/palantir/go-baseapp?status.svg)](http://godoc.org/github.com/palantir/go-baseapp)

A minimal, not-quite-a-framework for building web applications on top of the
standard library. It provides:

- A selection of dependencies for logging, metrics, and routing that fit well
  with the standard library
- A standard configuration type
- A basic configurable server type
- A default (but optional) middleware stack

This doesn't take the place of frameworks like [echo][], [gin][], or others,
but if you prefer to stay close to the standard library for simple
applications, `go-baseapp` will save you time when starting new projects.

[echo]: https://echo.labstack.com/
[gin]: https://gin-gonic.github.io/gin/

## Usage

Create a `baseapp.Server` object, register your handlers, and start the server:


```go
func main() {
    config := baseapp.HTTPConfig{
        Address: "127.0.0.1",
        Port:    8000,
    }
    loggingConfig := baseapp.LoggingConfig{
        Pretty: true,
        Level: "debug",
    }

    logger := baseapp.NewLogger(loggingConfig)

    // create a server with default options and no metrics prefix
    server, err := baseapp.NewServer(config, baseapp.DefaultParams(logger, "")...)
    if err != nil {
        panic(err)
    }

    // register handlers
    server.Mux().Handle(pat.Get("/hello"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        baseapp.WriteJSON(w, http.StatusOK, map[string]string{
            "message": fmt.Sprintf("Hello, %s!", r.FormValue("name")),
        })
    }))

    // start the server (blocking)
    server.Start()
}
```

### Example

The [example package](example/main.go) provides a full server that responds to
`GET /api/message` requests with a static message from the configuration.

You can start the server using:

    ./godelw dep
    ./godelw run example

Navigate to `http://localhost:8000/api/message` to see the output.

## Features and Opinions

`go-baseapp` was designed and is maintained by Palantir's developer tools team.
When developing this project, we had the following in mind:

1. Minimize code and knowledge beyond the standard library. Because we support
   many applications that have many contributors, we want to minimize the time
   people spend learning frameworks before being able to add new features or
   develop new services. To this end, we've used the standard library where we
   can and picked small dependencies with obvious APIs when we need more
   features.

3. Allow easy extension. Our opinions in this library form a starting point,
   but it should be possible to ignore any of them and add your solutions if
   you disagree with something or it doesn't work in a specific situation.

4. Provide ops-friendly defaults. Our default configuration enables things like
   request IDs, request and runtime metrics, nice error formatting, and panic
   recovery.  These are all easy to implement, but are also easy to forget when
   setting up a new project.

Notably, performance is not a major concern. While we try to avoid decisions
that are obviously not performant, our request scale is usually low and it
makes more sense to optimize on the factors above.

### Dependencies

Our selected dependencies are ultimately somewhat arbitrary, but have worked
well for us and do not include transitive dependencies of their own:

- [rs/zerolog](https://github.com/rs/zerolog) for logging. We like that it has
  an easy-to-use API, built-in support for `context.Context`, and helpful
  `net/http` integration via the `hlog` package.
- [rcrowley/go-metrics](https://github.com/rcrowley/go-metrics) for metrics. We
  like that it supports many metrics types, isn't coupled to a specific publish
  method, and has existing integrations with many monitoring and aggregation
  tools.
- [goji.io](http://goji.io/) for routing. We like that it provides a minimal
  API, supports middleware, and integrates nicely with `context.Context`.
- [bluekeyes/hatpear](https://github.com/bluekeyes/hatpear) for error
  aggregation. We like that it allows returning errors from HTTP handlers in a
  way that integrates well with `net/http`.

### Server Options

The default server configuration (`baseapp.DefaultParams`) does the following:

- Sets a logger
- Creates a metrics registry with an optional prefix
- Adds the default middleware to all routes (see below)
- Configures log timestamps to use UTC and nanosecond precision
- Improves the formatting of logged errors
- Enables Go runtime metrics collection

If you only want some of these options, provide your own set of parameters with
only the parts you want; all of the components are exported parts of this
library or dependencies.

### Graceful Shutdown

`go-baseapp` can be optionally configured to gracefully stop the running server by handling SIGINT and SIGTERM.

The parameter `ShutdownWaitTime` on the `baseapp.HTTPConfig` struct enables graceful shutdown, and
also informs the server how long to wait during the shutdown process before terminating.

### Middleware

The default middleware stack (`baseapp.DefaultMiddleware`) does the following:

- Adds a logger to the request context
- Adds a metrics registry to the request context
- Generates an ID for all requests and sets the `X-Request-ID` header
- Logs and emits metrics for all requests
- Handles errors returned by individual route handlers
- Recovers from panics in individual route handlers

If you only want some of these features, create your own middleware stack
selecting the parts you want; all of the components are exported parts of this
library or dependencies.

### Metrics

If enabled, the server emits the following metrics:

| metric name | type | definition |
| ----------- | ---- | ---------- |
| `server.requests` | `counter` | the count of requests handled by the server |
| `server.requests.2xx` | `counter` | like `server.requests`, but only counting 2XX status codes |
| `server.requests.3xx` | `counter` | like `server.requests`, but only counting 3XX status codes |
| `server.requests.4xx` | `counter` | like `server.requests`, but only counting 4XX status codes |
| `server.requests.5xx` | `counter` | like `server.requests`, but only counting 5XX status codes |
| `server.goroutines` | `gauge` | the number of active goroutines |
| `server.mem.used` | `gauge` | the amount of memory used by the process |

The `baseapp/datadog` package provides an easy way to publish metrics to
Datadog. Other aggregators can be configured with custom code in a similar way.

## Contributing

Contributions and issues are welcome. For new features or large contributions,
we prefer discussing the proposed change on a GitHub issue prior to a PR.

New functionality should avoid adding new dependencies if possible and should
be broadly useful. Feature requests that are specific to certain uses will
likely be declined unless they can be redesigned to be generic or optional.

Before submitting a pull request, please run tests and style checks:

```
./godelw verify
```

## License

This library is made available under the [Apache 2.0 License](http://www.apache.org/licenses/LICENSE-2.0).
