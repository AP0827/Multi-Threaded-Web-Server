package core

import (
	mtwshttp "MTWS/http"
	"strings"
)

type HandlerFunc func(*ResponseWriter, *mtwshttp.Request)

type prefixRoute struct {
	prefix  string
	handler HandlerFunc
}

type Router struct {
	routes       map[string]HandlerFunc
	prefixRoutes []prefixRoute
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]HandlerFunc),
	}
}

func (r *Router) Handle(path string, handler HandlerFunc) {
	if r == nil || path == "" || handler == nil {
		return
	}

	r.routes[path] = handler
}

func (r *Router) HandlePrefix(prefix string, handler HandlerFunc) {
	if r == nil || prefix == "" || handler == nil {
		return
	}

	r.prefixRoutes = append(r.prefixRoutes, prefixRoute{
		prefix:  prefix,
		handler: handler,
	})
}

func (r *Router) ServeHTTP(w *ResponseWriter, req *mtwshttp.Request) {
	if r == nil || w == nil || req == nil {
		return
	}

	path, _, _ := strings.Cut(req.Path(), "?")
	handler, ok := r.routes[path]
	if !ok {
		for _, route := range r.prefixRoutes {
			if strings.HasPrefix(path, route.prefix) {
				route.handler(w, req)
				return
			}
		}

		_ = w.WriteText(StatusNotFound, StatusNotFound.Body)
		return
	}

	handler(w, req)
}
