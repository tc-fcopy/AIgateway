package reverse_proxy

import (
	"errors"
	"gateway/middleware"
	"gateway/reverse_proxy/load_balance"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const (
	proxyErrorHeader       = "X-GW-Proxy-Error"
	proxyErrorDetailHeader = "X-GW-Proxy-Error-Detail"
)

func NewLoadBalanceReverseProxy(c *gin.Context, lb load_balance.LoadBalance, trans *http.Transport) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		nextAddr := ""
		if v, ok := c.Get("proxy_target"); ok {
			if addr, ok := v.(string); ok {
				nextAddr = strings.TrimSpace(addr)
			}
		}

		var lbErr error
		if nextAddr == "" {
			nextAddr, lbErr = lb.Get(req.URL.String())
		}
		if lbErr != nil || strings.TrimSpace(nextAddr) == "" {
			code := "no_upstream"
			if lbErr != nil && !errors.Is(lbErr, load_balance.ErrNoUpstream) {
				code = "lb_error"
			}
			setProxyError(req, code, lbErr)
			clearTarget(req)
			return
		}

		target, err := url.Parse(nextAddr)
		if err != nil || target.Scheme == "" || target.Host == "" {
			setProxyError(req, "invalid_upstream", err)
			clearTarget(req)
			return
		}

		req.Header.Del(proxyErrorHeader)
		req.Header.Del(proxyErrorDetailHeader)

		targetQuery := target.RawQuery
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		req.Host = target.Host
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "user-agent")
		}
	}

	modifyFunc := func(resp *http.Response) error {
		if strings.Contains(resp.Header.Get("Connection"), "Upgrade") {
			return nil
		}
		return nil
	}

	errFunc := func(w http.ResponseWriter, r *http.Request, err error) {
		if code := r.Header.Get(proxyErrorHeader); code != "" {
			detail := r.Header.Get(proxyErrorDetailHeader)
			switch code {
			case "no_upstream":
				middleware.ResponseError(c, 2002, errors.New(defaultProxyError(detail, "no upstream available")))
				return
			case "invalid_upstream":
				middleware.ResponseError(c, 2002, errors.New(defaultProxyError(detail, "invalid upstream address")))
				return
			case "lb_error":
				middleware.ResponseError(c, 2002, errors.New(defaultProxyError(detail, "load balance error")))
				return
			default:
				middleware.ResponseError(c, 2002, errors.New(defaultProxyError(detail, "reverse proxy error")))
				return
			}
		}
		middleware.ResponseError(c, 999, err)
	}
	return &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: modifyFunc,
		ErrorHandler:   errFunc,
		Transport:      trans,
	}
}

func setProxyError(req *http.Request, code string, err error) {
	req.Header.Set(proxyErrorHeader, code)
	if err != nil {
		req.Header.Set(proxyErrorDetailHeader, err.Error())
	}
}

func defaultProxyError(detail, fallback string) string {
	if strings.TrimSpace(detail) != "" {
		return detail
	}
	return fallback
}

func clearTarget(req *http.Request) {
	req.URL.Scheme = ""
	req.URL.Host = ""
	req.Host = ""
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
