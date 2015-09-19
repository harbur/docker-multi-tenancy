package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type Client struct {
	endpoint       string
	endpointURL    *url.URL
	unixHTTPClient *http.Client
	dialer         func(string, string) (net.Conn, error)
}

// NewVersionedClient returns a Client instance ready for communication with
// the given server endpoint
func NewClient(endpoint string) (*Client, error) {
	u, err := url.Parse(endpoint)

	if err != nil {
		return nil, ErrInvalidEndpoint
	}

	d := net.Dialer{}
	dialFunc := func(network, addr string) (net.Conn, error) {
		return d.Dial("unix", u.Path)
	}
	unixHTTPClient := &http.Client{
		Transport: &http.Transport{
			Dial: dialFunc,
		},
	}

	return &Client{
		unixHTTPClient: unixHTTPClient,
		endpoint:       endpoint,
		endpointURL:    u,
		dialer:         dialFunc,
	}, nil
}

// getFakeUnixURL returns the URL needed to make an HTTP request over a UNIX
// domain socket to the given path.
func (c *Client) getFakeUnixURL(path string) string {
	u := *c.endpointURL // Copy.

	// Override URL so that net/http will not complain.
	u.Scheme = "http"
	u.Host = "unix.sock" // Doesn't matter what this is - it's not used.
	u.Path = ""

	urlStr := strings.TrimRight(u.String(), "/")

	return fmt.Sprintf("%s%s", urlStr, path)
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	var u string

	httpClient := c.unixHTTPClient
	u = c.getFakeUnixURL(path)

	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)

	if logResponse {
		if respBytes, err := httputil.DumpResponse(resp, true); err == nil {
			log.Printf("Response:\n%v\n", string(respBytes))
		}

	}
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return nil, ErrConnectionRefused
		}
		return resp, err
	}

	return resp, nil
}
