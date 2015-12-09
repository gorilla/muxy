package muxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/muxy"
	"golang.org/x/net/context"
)

func New(options ...func(*matcher)) *muxy.Router {
	// TODO: options variadic argument (to set NotFound, strict slashes etc).
	m := &matcher{}
	for _, o := range options {
		o(m)
	}
	return muxy.New(m)
}

type matcher struct {
	// TODO...
}

func (m *matcher) Route(pattern string) (*muxy.Route, error) {
	// TODO...
	return nil, nil
}

func (m *matcher) Match(c context.Context, r *http.Request) (context.Context, muxy.Handler) {
	// TODO...
	return nil, nil
}

func (m *matcher) URL(r *muxy.Route, values map[string]string) (*url.URL, error) {
	// TODO...
	return nil, nil
}

// methodHandler returns the handler registered for the given HTTP method.
func methodHandler(handlers map[string]muxy.Handler, method string) muxy.Handler {
	if handlers == nil || len(handlers) == 0 {
		return nil
	}
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
func allowHandler(handlers map[string]muxy.Handler, code int) muxy.Handler {
	allowed := make([]string, len(handlers)+1)
	allowed[0] = "OPTIONS"
	i := 1
	for m, _ := range handlers {
		if m != "" && m != "OPTIONS" {
			allowed[i] = m
			i++
		}
	}
	return muxy.HandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Allow", strings.Join(allowed[:i], ", "))
		w.WriteHeader(code)
		fmt.Fprintln(w, code, http.StatusText(code))
	})
}
