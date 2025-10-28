package httpUtils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type HTTPClient interface {
	GET(ctx context.Context, url string, header map[string]interface{}) (status int, respBody []byte, err error)
	POST(ctx context.Context, url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, err error)
	PUT(ctx context.Context, url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, err error)
	PATCH(ctx context.Context, url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, err error)
	DELETE(ctx context.Context, url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, err error)
}

type httpClient struct {
	debug bool
	c     *http.Client
}

var (
	hcOnce sync.Once
	hc     *httpClient
)

func NewHTTPClient() HTTPClient {
	hcOnce.Do(func() {
		hc = &httpClient{
			c: &http.Client{
				Timeout: time.Second * 3,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			},
		}
	})
	return hc
}

func NewHTTPClientWithDebug(debug bool) HTTPClient {
	hcOnce.Do(func() {
		hc = &httpClient{
			debug: debug,
			c: &http.Client{
				Timeout: time.Second * 3,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			},
		}
	})
	return hc
}

func (hc *httpClient) GET(ctx context.Context, url string, header map[string]interface{}) (status int, respBody []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}

	hc.setHeader(req, header)

	return hc.do(req)
}

func (hc *httpClient) POST(ctx context.Context, url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, err error) {
	var dataByte []byte
	switch data := body.(type) {
	case []byte:
		dataByte = data
	case string:
		dataByte = []byte(data)
	default:
		dataByte, err = json.Marshal(data)
		if err != nil {
			return
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(dataByte))
	if err != nil {
		return
	}

	hc.setHeader(req, header)

	return hc.do(req)
}

func (hc *httpClient) PUT(ctx context.Context, url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, err error) {
	var dataByte []byte
	switch data := body.(type) {
	case []byte:
		dataByte = data
	case string:
		dataByte = []byte(data)
	default:
		dataByte, err = json.Marshal(data)
		if err != nil {
			return
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(dataByte))
	if err != nil {
		return
	}

	hc.setHeader(req, header)

	return hc.do(req)
}

func (hc *httpClient) PATCH(ctx context.Context, url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, err error) {
	var dataByte []byte
	switch data := body.(type) {
	case []byte:
		dataByte = data
	case string:
		dataByte = []byte(data)
	default:
		dataByte, err = json.Marshal(data)
		if err != nil {
			return
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewBuffer(dataByte))
	if err != nil {
		return
	}

	hc.setHeader(req, header)

	return hc.do(req)
}

func (hc *httpClient) DELETE(ctx context.Context, url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, err error) {
	var dataByte []byte
	switch data := body.(type) {
	case []byte:
		dataByte = data
	case string:
		dataByte = []byte(data)
	default:
		dataByte, err = json.Marshal(data)
		if err != nil {
			return
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, bytes.NewBuffer(dataByte))
	if err != nil {
		return
	}

	hc.setHeader(req, header)

	return hc.do(req)
}

func (hc *httpClient) do(req *http.Request) (status int, respBody []byte, err error) {
	var reqBody []byte
	if hc.debug {
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			return
		}
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	}

	resp, err := hc.c.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if hc.debug {
		hc.printDebugInfo(req, reqBody, resp.StatusCode, respBody)
	}

	return resp.StatusCode, respBody, nil
}

func (hc *httpClient) setHeader(req *http.Request, header map[string]interface{}) {
	for k, v := range header {
		req.Header.Set(k, v.(string))
	}
}

func (hc *httpClient) setCookie(req *http.Request, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
}

func (hc *httpClient) printDebugInfo(req *http.Request, reqBody []byte, statusCode int, respBody []byte) {
	var buf bytes.Buffer

	// Request Info
	buf.WriteString("\n" + strings.Repeat("=", 61) + "\n")

	buf.WriteString("[Request]\n")
	buf.WriteString(fmt.Sprintf("Method: %s\n", req.Method))
	buf.WriteString(fmt.Sprintf("URL: %s\n", req.URL.String()))
	buf.WriteString(fmt.Sprintf("Headers: %v\n", req.Header))
	if len(reqBody) > 0 {
		// 尝试格式化JSON，如果失败则直接打印
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, reqBody, "", "  "); err == nil {
			buf.WriteString("Body:\n")
			buf.Write(prettyJSON.Bytes())
			buf.WriteString("\n")
		} else {
			buf.WriteString(fmt.Sprintf("Body: %s\n", string(reqBody)))
		}
	}

	// Response Info
	buf.WriteString("\n[Response]\n")
	buf.WriteString(fmt.Sprintf("Status Code: %d\n", statusCode))
	if len(respBody) > 0 {
		// 尝试格式化JSON，如果失败则直接打印
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, respBody, "", "  "); err == nil {
			buf.WriteString("Body:\n")
			buf.Write(prettyJSON.Bytes())
			buf.WriteString("\n")
		} else {
			buf.WriteString(fmt.Sprintf("Body: %s\n", string(respBody)))
		}
	}

	buf.WriteString("\n" + strings.Repeat("=", 61) + "\n")

	// 一次性输出，避免并发场景下日志混乱
	fmt.Print(buf.String())
}
