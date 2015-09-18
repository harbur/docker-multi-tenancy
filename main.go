package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"net/url"
	"errors"
	"os"
	"io"
	"io/ioutil"
	"log"
)

const userAgent = "docker-multi-tenancy"

var (
	// ErrInvalidEndpoint is returned when the endpoint is not a valid HTTP URL.
	ErrInvalidEndpoint = errors.New("invalid endpoint")

	ErrConnectionRefused = errors.New("Connection refused")
)


type Client struct{
	endpoint            string
	endpointURL         *url.URL
	unixHTTPClient      *http.Client
	dialer              func(string, string) (net.Conn, error)
}


// Error represents failures in the API. It represents a failure from the API.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.Status, e.Message)
}


var (
	localAddrString  = ":9000"
	unixDockerSocket = "unix:///var/run/docker.sock"

)



func dockerRequestHandler(transformers map[string]func(r *http.Request)) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Recieved methos ", r.Method, " url", r.URL)
		c, err := newClient(unixDockerSocket)

		if err != nil {
			w.Write([]byte("Error"))
			return
		}

		for k, f := range transformers{
			fmt.Println("Applixing expresion ", k)
			f(r)
		}

		// TODO needs to accept POST content
		resp, err := c.do( r.Method,  r.URL.String())

		if err != nil {
			w.Write([]byte("Error"))
			return
		}

		defer resp.Body.Close()

		content, _ := ioutil.ReadAll(resp.Body)

		w.Write(content)
	}
	return http.HandlerFunc(fn)
}

func main(){

	fmt.Println("Starting multi-tenancy proxy")

	f := func(r *http.Request){
		fmt.Println("Modifiy somehow the request")
	}

	transformers := make(map[string]func(r *http.Request))

	transformers["*"] = f

	http.ListenAndServe(localAddrString, dockerRequestHandler(transformers))

}

func forward(localConn net.Conn) {
	c, err := newClient(unixDockerSocket)

	sockerCon, err := c.dialer("unix", c.endpointURL.Path)

	if err != nil {
		fmt.Println("Error dialing")
		os.Exit(-1)
	}

	// Copy localConn.Reader to sshConn.Writer
	go func() {
		_, err = io.Copy(sockerCon, localConn)
		if err != nil {
			log.Fatalf("io.Copy failed: %v", err)
		}
	}()

	// Copy sshConn.Reader to localConn.Writer
	go func() {
		_, err = io.Copy(localConn, sockerCon)
		if err != nil {
			log.Fatalf("io.Copy failed: %v", err)
		}
	}()
}


// NewVersionedClient returns a Client instance ready for communication with
// the given server endpoint
func newClient(endpoint string) (*Client, error) {
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
		unixHTTPClient:      unixHTTPClient,
		endpoint:            endpoint,
		endpointURL:         u,
		dialer:				 dialFunc,
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


func (c *Client) do(method, path string) (*http.Response, error) {
	var params io.Reader
	var u string

	httpClient := c.unixHTTPClient
	u = c.getFakeUnixURL(path)


	req, err := http.NewRequest(method, u, params)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	if method == "POST" {
		req.Header.Set("Content-Type", "plain/text")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return nil, ErrConnectionRefused
		}
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, newError(resp)
	}
	return resp, nil
}

func newError(resp *http.Response) *Error {
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &Error{Status: resp.StatusCode, Message: fmt.Sprintf("cannot read body, err: %v", err)}
	}
	return &Error{Status: resp.StatusCode, Message: string(data)}
}
