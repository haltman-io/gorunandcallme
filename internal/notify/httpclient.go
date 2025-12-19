package notify

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/haltman-io/gorunandcallme/internal/config"
	"golang.org/x/net/proxy"
)

func NewHTTPClient(tcfg config.TransportConfig) (*http.Client, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: tcfg.InsecureTLS}, // intended by user (-k)
		Proxy:           http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   20 * time.Second,
			KeepAlive: 20 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if tcfg.NoProxyEnv {
		tr.Proxy = nil
	}

	if tcfg.Proxy != "" {
		u, err := url.Parse(tcfg.Proxy)
		if err != nil {
			return nil, err
		}
		switch strings.ToLower(u.Scheme) {
		case "http", "https":
			tr.Proxy = http.ProxyURL(u)
			if tcfg.ProxyAuth != "" {
				// Basic auth for proxy CONNECT (best-effort)
				tr.ProxyConnectHeader = proxyAuthHeader(tcfg.ProxyAuth)
			}
		case "socks5", "socks5h":
			dialer, err := socks5Dialer(u, tcfg.ProxyAuth)
			if err != nil {
				return nil, err
			}
			tr.Proxy = nil
			tr.DialContext = dialer.(proxy.ContextDialer).DialContext
		default:
			return nil, errors.New("unsupported proxy scheme (use http(s):// or socks5://)")
		}
	}

	return &http.Client{
		Timeout: 25 * time.Second,
		Transport: tr,
	}, nil
}

func proxyAuthHeader(userpass string) http.Header {
	h := http.Header{}
	// We set Proxy-Authorization only for CONNECT flows where supported.
	// Note: Go's Transport does not expose full control for every proxy auth scenario.
	encoded := basicAuth(userpass)
	if encoded != "" {
		h.Set("Proxy-Authorization", "Basic "+encoded)
	}
	return h
}

func basicAuth(userpass string) string {
	parts := strings.SplitN(userpass, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	// Avoid importing encoding/base64 into many files; keep local.
	const table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	raw := []byte(parts[0] + ":" + parts[1])

	// minimal base64
	var out strings.Builder
	for i := 0; i < len(raw); i += 3 {
		var b [3]byte
		n := 0
		for j := 0; j < 3 && i+j < len(raw); j++ {
			b[j] = raw[i+j]
			n++
		}
		val := uint(b[0])<<16 | uint(b[1])<<8 | uint(b[2])
		for j := 0; j < 4; j++ {
			idx := (val >> uint(18-6*j)) & 0x3F
			out.WriteByte(table[idx])
		}
		if n < 3 {
			outStr := out.String()
			out.Reset()
			out.WriteString(outStr[:len(outStr)-(3-n)])
			for k := 0; k < (3 - n); k++ {
				out.WriteByte('=')
			}
		}
	}
	return out.String()
}

func socks5Dialer(u *url.URL, userpass string) (proxy.Dialer, error) {
	var auth *proxy.Auth
	if userpass != "" {
		parts := strings.SplitN(userpass, ":", 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid --proxy-auth format, expected user:pass")
		}
		auth = &proxy.Auth{User: parts[0], Password: parts[1]}
	}
	addr := u.Host
	if addr == "" {
		return nil, errors.New("invalid SOCKS5 proxy address")
	}
	return proxy.SOCKS5("tcp", addr, auth, proxy.Direct)
}
