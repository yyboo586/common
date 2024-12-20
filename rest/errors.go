package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	InternalServerError = NewHTTPError(http.StatusInternalServerError, "Internal Server Error, Please try again later", nil)
	BadRequest          = NewHTTPError(http.StatusBadRequest, "Bad Request, Please check your request", nil)
	Unauthorized        = NewHTTPError(http.StatusUnauthorized, "Unauthorized, Please check your token", nil)
	Forbidden           = NewHTTPError(http.StatusForbidden, "Forbidden, You are not allowed to access this resource", nil)
	NotFound            = NewHTTPError(http.StatusNotFound, "Not Found, The requested resource was not found", nil)
	MethodNotAllowed    = NewHTTPError(http.StatusMethodNotAllowed, "Method Not Allowed", nil)
	Conflict            = NewHTTPError(http.StatusConflict, "Conflict", nil)
	TooManyRequests     = NewHTTPError(http.StatusTooManyRequests, "Too Many Requests", nil)
)

type HTTPError struct {
	Code    int
	Message string
	Details map[string]interface{}
}

func NewHTTPError(code int, msg string, details map[string]interface{}) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: msg,
		Details: details,
	}
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("错误码: %d, 错误信息: %v, 详细错误信息: %v\n", e.Code, e.Message, e.Details)
}

func (e *HTTPError) StatusCode() int {
	return e.Code
}

func ReplyError(c *gin.Context, err error) {
	var code int
	var body []byte

	switch e := err.(type) {
	case *HTTPError:
		code = e.StatusCode()
		body, _ = json.Marshal(e)
	default:
		code = http.StatusInternalServerError
		body = []byte(err.Error())
	}

	// 设置Header前先设置响应码
	c.Writer.WriteHeader(code)
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Write(body)
}

func ReplyOK(c *gin.Context, statusCode int, data interface{}) {
	var body []byte

	if data != nil {
		body, _ = json.Marshal(data)
	}

	c.Writer.WriteHeader(statusCode)
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Write(body)
}
