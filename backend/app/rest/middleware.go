package rest

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg4go/rewrite"
)

// JSON is a map alias, just for convenience
type JSON map[string]interface{}

type contextKey string

// Rewrite middleware with from->to rule. Supports regex (like nginx) and prevents multiple rewrites
func Rewrite(from, to string) func(http.Handler) http.Handler {
	rule, err := rewrite.NewRule(from, to)
	if err != nil {
		log.Printf("[WARN] can't parse rewrite rule %s - > %s, %s", from, to, err)
	}

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			ctx := r.Context()
			// prevent double rewrites
			if ctx != nil {
				if _, ok := ctx.Value(contextKey("rewrite")).(bool); ok {
					h.ServeHTTP(w, r)
					return
				}
			}

			if err == nil {
				rule.Rewrite(r)
				ctx = context.WithValue(ctx, contextKey("rewrite"), true)
				r = r.WithContext(ctx)
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

// LoggerFlag type
type LoggerFlag int

// logger flags enum
const (
	LogAll LoggerFlag = iota
	LogBody
)
const maxBody = 1024

var reMultWhtsp = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

// Logger middleware prints http log. Customized by set of LoggerFlag
func Logger(flags ...LoggerFlag) func(http.Handler) http.Handler {

	inFlags := func(f LoggerFlag) bool {
		for _, flg := range flags {
			if flg == LogAll || flg == f {
				return true
			}
		}
		return false
	}

	f := func(h http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, 1)

			body := func() (result string) {
				if inFlags(LogBody) {
					if content, err := ioutil.ReadAll(r.Body); err == nil {
						result = string(content)
						r.Body = ioutil.NopCloser(bytes.NewReader(content))

						if len(result) > 0 {
							result = strings.Replace(result, "\n", " ", -1)
							result = reMultWhtsp.ReplaceAllString(result, " ")
						}

						if len(result) > maxBody {
							result = result[:maxBody] + "..."
						}
					}
				}
				return result
			}()

			t1 := time.Now()
			defer func() {
				t2 := time.Now()

				q := r.URL.String()
				if qun, err := url.QueryUnescape(q); err == nil {
					q = qun
				}
				// hide id and pin
				if strings.Contains(q, "/api/v1/message/") {
					elems := strings.Split(q, "/")
					if len(elems) >= 5 && len(elems[4]) >= 20 {
						q = fmt.Sprintf("/api/v1/message/%s/*****", elems[4][:19])
					}
				}
				log.Printf("[INFO] REST %s - %s - %s - %d (%d) - %v %s",
					r.Method, q, strings.Split(r.RemoteAddr, ":")[0],
					ww.Status(), ww.BytesWritten(), t2.Sub(t1), body)
			}()

			h.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}

	return f
}
