package MiddleWare

import "github.com/gogf/gf/v2/net/ghttp"

func CORS(r *ghttp.Request) {
	corsOptions := r.Response.DefaultCORSOptions()
	r.Response.CORS(corsOptions)
	r.Middleware.Next()
}
