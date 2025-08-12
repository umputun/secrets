## tollbooth_chi

[Chi](https://github.com/pressly/chi) middleware for rate limiting HTTP requests.

## Five Minutes Tutorial

### With Tollbooth Version 8 and Higher

You won't need to use this wrapper anymore and can use `tollbooth.HTTPMiddleware` directly.

```go
package main

import (
	"net/http"

	"github.com/didip/tollbooth/v7"
	"github.com/pressly/chi"
)

func main() {
	// Create a limiter struct.
	limiter := tollbooth.NewLimiter(1, nil)

	r := chi.NewRouter()

	lmt.SetIPLookup(limiter.IPLookup{
		Name:           "X-Real-IP",
		IndexFromRight: 0,
	})

	// Use HTTPMiddleware directly for easier integration with chi.
	r.Use(tollbooth.HTTPMiddleware(limiter))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})

	http.ListenAndServe(":12345", r)
}

```

### With Tollbooth version 7 and below

```
package main

import (
    "net/http"
    
    "github.com/didip/tollbooth/v7"
    "github.com/didip/tollbooth_chi"
    "github.com/pressly/chi"
)

func main() {
    // Create a limiter struct.
    limiter := tollbooth.NewLimiter(1, nil)

    r := chi.NewRouter()

    r.Use(tollbooth_chi.LimitHandler(limiter))

    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, world!"))
    })

    http.ListenAndServe(":12345", r)
}
```
