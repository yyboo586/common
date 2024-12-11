package httpUtils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type OAuth2HTTPClient interface {
	GET(url string, header map[string]interface{}, cookies []*http.Cookie) (status int, respBody []byte, respHeader http.Header, err error)
	POST(url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, respHeader http.Header, err error)
	PUT(url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, respHeader http.Header, err error)
}

type oauth2HTTPClient struct {
	c *http.Client
}

var (
	hcOnce sync.Once
	hc     *oauth2HTTPClient
)

func NewOAuth2HTTPClient() OAuth2HTTPClient {
	hcOnce.Do(func() {
		hc = &oauth2HTTPClient{
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

func (o2Client *oauth2HTTPClient) GET(url string, header map[string]interface{}, cookies []*http.Cookie) (status int, respBody []byte, respHeader http.Header, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	o2Client.setHeader(req, header)
	o2Client.setCookie(req, cookies)

	return o2Client.do(req)
}

func (o2Client *oauth2HTTPClient) POST(url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, respHeader http.Header, err error) {
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

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(dataByte))
	if err != nil {
		return
	}

	o2Client.setHeader(req, header)

	return o2Client.do(req)
}

func (o2Client *oauth2HTTPClient) PUT(url string, header map[string]interface{}, body interface{}) (status int, respBody []byte, respHeader http.Header, err error) {
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

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(dataByte))
	if err != nil {
		return
	}

	o2Client.setHeader(req, header)

	return o2Client.do(req)
}

func (o2Client *oauth2HTTPClient) do(req *http.Request) (status int, respBody []byte, respHeader http.Header, err error) {
	resp, err := o2Client.c.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return resp.StatusCode, respBody, resp.Header, nil
}

func (o2Client *oauth2HTTPClient) setHeader(req *http.Request, header map[string]interface{}) {
	for k, v := range header {
		req.Header.Set(k, v.(string))
	}
}

func (o2Client *oauth2HTTPClient) setCookie(req *http.Request, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
}

func GetChallengeFromHeader(h http.Header, splitStr string) (string, error) {
	location := h.Get("Location")
	arr := strings.Split(location, splitStr)
	if len(arr) < 2 {
		return "", fmt.Errorf("invalid location header: %s", location)
	}

	return arr[1], nil
}

func GetCodeFromHeader(h http.Header, splitStr string) (string, error) {
	location := h.Get("Location")
	arr := strings.Split(location, splitStr)
	if len(arr) < 2 {
		return "", fmt.Errorf("invalid location header: %s", location)
	}

	arr1 := strings.Split(arr[1], "&")

	return arr1[0], nil
}

func ReadSetCookies(h http.Header) []*http.Cookie {
	cookieCount := len(h["Set-Cookie"])
	if cookieCount == 0 {
		return []*http.Cookie{}
	}
	cookies := make([]*http.Cookie, 0, cookieCount)
	for _, line := range h["Set-Cookie"] {
		if cookie, err := http.ParseSetCookie(line); err == nil {
			cookies = append(cookies, cookie)
		}
	}
	return cookies
}
