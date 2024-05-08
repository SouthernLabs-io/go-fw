# Go Framework
This is a go framework to build servers.

It is implemented with:
 - HTTP: [Gin Web Framework](https://github.com/gin-gonic/gin)
 - ORM: [GORM](https://github.com/go-gorm/gorm)
 - DI: [Uber Fx](https://github.com/uber-go/fx)
 - Configuration: [Viper](https://github.com/spf13/viper) 
 - CLI: [Cobra](https://github.com/spf13/cobra) 
 - Monitoring: [DataDog](https://github.com/DataDog/dd-trace-go)

This is an evolution of the ideas found in: https://github.com/dipeshdulal/clean-gin and inspiration
from: https://www.dropwizard.io/en/latest/

## Getting Started

Please make sure to familiarize yourself with each one of the libs/frameworks under use. 

Now that you are ready, lets start by creating a simple server.

Create a new go module:
```shell
mkdir my-server
cd my-server
go mod init my-server
```

Now import this framework:
```shell
go get github.com/southernlabs-io/go-fw
```

Let's create a new `main.go` file that will set up the server:
```go
package main

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/bootstrap"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/panich"
	"github.com/southernlabs-io/go-fw/rest/middleware"
)

func main() {
	defer panich.DeferredPanicToLogAndExit()
	var deps = fx.Options(
		// middlewares
		middleware.RequestLoggerModule,
	)
	err := bootstrap.NewAppWithServe(deps).Execute()
	if err != nil {
		panic(errors.NewUnknownf("failed to execute with error: %w", err))
	}
}
```

Now let's create a config.yaml file:
```yaml
name: simple-server

env:
  name: local
  type: local
log:
  level: debug

httpServer:
  port: 8080
  bindAddress:
  basePath: /api/v1/
  reqLoggerExcludes: [ "/health", "/ready" ]

database:
  user: postgres
  pass: <secret> # It will be loaded from a SecretManager. It can also be overridden by an env var for development
  host: localhost
  port: 5432
```

Now we can create a `.env` file for use locally:
```shell    
DATABASE_PASS=postgres
```

Now let's run it without building it:
```shell    
 go run main.go app:server
```

To build it and run it:
```shell
# the following flags will set version info for the compiled binary:
 go build \
  -ldflags="-s -w \
  -X github.com/southernlabs-io/go-fw/version.BuildTime=$(date -u '+%Y-%m-%d_%I:%M:%S%p') \
  -X github.com/southernlabs-io/go-fw/version.Commit=$(git rev-parse HEAD) \
  -X github.com/southernlabs-io/go-fw/version.Release=$(git describe --tags --always --dirty)" \
  -o my-server

./my-server app:server
```

## Docs
- Configuration: [CONFIG.md](CONFIG.md)