package tokenUtils

import (
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/text/gstr"
)

// Middleware 绑定group
func (m *Token) Middleware(group *ghttp.RouterGroup) error {
	group.Middleware(m.authMiddleware)
	return nil
}

func (m *Token) authMiddleware(r *ghttp.Request) {
	_, err := m.Parse(r)
	if err != nil {
		r.Response.WriteJson(g.Map{
			"code":    401,
			"message": err.Error(),
		})
		return
	}
	r.Middleware.Next()
}

func (m *Token) AuthPath(urlPath string) bool {
	// 去除后斜杠
	if strings.HasSuffix(urlPath, "/") {
		urlPath = gstr.SubStr(urlPath, 0, len(urlPath)-1)
	}

	// 排除路径处理，到这里nextFlag为true
	for _, excludePath := range m.ExcludePaths {
		tmpPath := excludePath
		// 前缀匹配
		if strings.HasSuffix(tmpPath, "/*") {
			tmpPath = gstr.SubStr(tmpPath, 0, len(tmpPath)-2)
			if gstr.HasPrefix(urlPath, tmpPath) {
				// 前缀匹配不拦截
				return false
			}
		} else {
			// 全路径匹配
			if strings.HasSuffix(tmpPath, "/") {
				tmpPath = gstr.SubStr(tmpPath, 0, len(tmpPath)-1)
			}
			if urlPath == tmpPath {
				// 全路径匹配不拦截
				return false
			}
		}
	}
	return true
}
