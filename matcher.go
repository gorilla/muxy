package muxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/context"
)

func NewPathMatcher() *PathMatcher {
	// TODO: options variadic argument (to set NotFound, strict slashes etc).
	return &PathMatcher{}
}

type PathMatcher struct {
	// TODO...
}

func (m *PathMatcher) Route(pattern string) (*Route, error) {
	// TODO...
	return nil, nil
}

func (m *PathMatcher) Match(r *http.Request) (Handler, map[string]string) {
	// TODO...
	return nil, nil
}

func (m *PathMatcher) URL(r *Route, values map[string]string) (*url.URL, error) {
	// TODO...
	return nil, nil
}

// methodHandler returns the handler registered for the given HTTP method.
func methodHandler(handlers map[string]Handler, method string) Handler {
	if h, ok := handlers[method]; ok {
		return h
	}
	switch method {
	case "OPTIONS":
		return allowHandler(handlers, 200)
	case "HEAD":
		if h, ok := handlers["GET"]; ok {
			return h
		}
		fallthrough
	default:
		if h, ok := handlers[""]; ok {
			return h
		}
	}
	return allowHandler(handlers, 405)
}

// allowHandler returns a handler that sets a header with the given
// status code and allowed methods.
func allowHandler(handlers map[string]Handler, code int) Handler {
	allowed := make([]string, len(handlers)+1)
	allowed[0] = "OPTIONS"
	i := 1
	for m, _ := range handlers {
		if m != "" && m != "OPTIONS" {
			allowed[i] = m
			i++
		}
	}
	return HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Allow", strings.Join(allowed[:i], ", "))
		w.WriteHeader(code)
		fmt.Fprintln(w, code, http.StatusText(code))
	})
}
