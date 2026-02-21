package cli

import "net/http"

func newHTTPClient(proxyFromEnv bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyFromEnv {
		transport.Proxy = http.ProxyFromEnvironment
	} else {
		transport.Proxy = nil
	}
	return &http.Client{
		Transport: transport,
		Timeout:   0,
	}
}
