package core

import mtwshttp "MTWS/http"

type HandlerFunc func(*ResponseWriter, *mtwshttp.Request)

type Router struct {
	routes map[string]HandlerFunc
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

func (r *Router) ServeHTTP(w *ResponseWriter, req *mtwshttp.Request) {
	if r == nil || w == nil || req == nil {
		return
	}

	handler, ok := r.routes[req.Path()]
	if !ok {
		writeErrorResponse(w.conn, req.Version(), StatusNotFound)
		return
	}

	handler(w, req)
}
