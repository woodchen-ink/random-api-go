package router

import (
	"net/http"
	"random-api-go/middleware"
)

type Router struct {
	mux *http.ServeMux
}

type Handler interface {
	Setup(r *Router)
}

func New() *Router {
	return &Router{
		mux: http.NewServeMux(),
	}
}

func (r *Router) Setup(h Handler) {
	// 静态文件服务
	fileServer := http.FileServer(http.Dir("/root/data/public"))
	r.mux.Handle("/", middleware.Chain(
		middleware.Recovery,
		middleware.MetricsMiddleware,
	)(fileServer))

	// 设置API路由
	h.Setup(r)
}

func (r *Router) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	r.mux.HandleFunc(pattern, handler)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}
