

## Stackdriver nozzle


### Getting started

* Install dependencies (using [Glide](https://glide.sh/))

```
glide install --strip-vendor
```

* Nozzle Configuration:
    * Copy `.envrc.template` to `.envrc` & fill in values (uses [direnv](http://direnv.net/))
    * Or export variables represented in `.envrc.template` into a shell
    * Or use the command line switches (see `go run main.go --help`)
* Stackdriver configuration:
    * Uses the [DefaultTokenSource](https://godoc.org/golang.org/x/oauth2/google#DefaultTokenSource) to authenticate, configure according to linked doc