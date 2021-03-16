package rest

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"regexp"
)

// Rewrite middleware with from->to rule. Supports regex (like nginx) and prevents multiple rewrites
// example: Rewrite(`^/sites/(.*)/settings/$`, `/sites/settings/$1`
func Rewrite(from, to string) func(http.Handler) http.Handler {
	reFrom := regexp.MustCompile(from)

	f := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			ctx := r.Context()

			// prevent double rewrites
			if ctx != nil {
				if _, ok := ctx.Value(contextKey("rewrite")).(bool); ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			if !reFrom.MatchString(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			rwr := path.Clean(reFrom.ReplaceAllString(r.URL.Path, to))
			u, e := url.Parse(rwr)
			if e != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Header.Set("X-Original-URL", r.URL.RequestURI())
			r.URL.Path = u.Path
			r.URL.RawPath = u.RawPath
			if u.RawQuery != "" {
				r.URL.RawQuery = u.RawQuery
			}
			ctx = context.WithValue(ctx, contextKey("rewrite"), true)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
	return f
}
