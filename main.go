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
	"time"
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

// APIImages represent an image returned in the ListImages call.
type APIImages struct {
	ID          string            `json:"Id" yaml:"Id"`
	RepoTags    []string          `json:"RepoTags,omitempty" yaml:"RepoTags,omitempty"`
	Created     int64             `json:"Created,omitempty" yaml:"Created,omitempty"`
	Size        int64             `json:"Size,omitempty" yaml:"Size,omitempty"`
	VirtualSize int64             `json:"VirtualSize,omitempty" yaml:"VirtualSize,omitempty"`
	ParentID    string            `json:"ParentId,omitempty" yaml:"ParentId,omitempty"`
	RepoDigests []string          `json:"RepoDigests,omitempty" yaml:"RepoDigests,omitempty"`
	Labels      map[string]string `json:"Labels,omitempty" yaml:"Labels,omitempty"`
}

// Port represents the port number and the protocol, in the form
// <number>/<protocol>. For example: 80/tcp.
type Port string

// Port returns the number of the port.
func (p Port) Port() string {
	return strings.Split(string(p), "/")[0]
}

// Mount represents a mount point in the container.
//
// It has been added in the version 1.20 of the Docker API, available since
// Docker 1.8.
type Mount struct {
	Source      string
	Destination string
	Mode        string
	RW          bool
}

// Config is the list of configuration options used when creating a container.
// Config does not contain the options that are specific to starting a container on a
// given host.  Those are contained in HostConfig
type Config struct {
	Hostname        string              `json:"Hostname,omitempty" yaml:"Hostname,omitempty"`
	Domainname      string              `json:"Domainname,omitempty" yaml:"Domainname,omitempty"`
	User            string              `json:"User,omitempty" yaml:"User,omitempty"`
	Memory          int64               `json:"Memory,omitempty" yaml:"Memory,omitempty"`
	MemorySwap      int64               `json:"MemorySwap,omitempty" yaml:"MemorySwap,omitempty"`
	CPUShares       int64               `json:"CpuShares,omitempty" yaml:"CpuShares,omitempty"`
	CPUSet          string              `json:"Cpuset,omitempty" yaml:"Cpuset,omitempty"`
	AttachStdin     bool                `json:"AttachStdin,omitempty" yaml:"AttachStdin,omitempty"`
	AttachStdout    bool                `json:"AttachStdout,omitempty" yaml:"AttachStdout,omitempty"`
	AttachStderr    bool                `json:"AttachStderr,omitempty" yaml:"AttachStderr,omitempty"`
	PortSpecs       []string            `json:"PortSpecs,omitempty" yaml:"PortSpecs,omitempty"`
	ExposedPorts    map[Port]struct{}   `json:"ExposedPorts,omitempty" yaml:"ExposedPorts,omitempty"`
	Tty             bool                `json:"Tty,omitempty" yaml:"Tty,omitempty"`
	OpenStdin       bool                `json:"OpenStdin,omitempty" yaml:"OpenStdin,omitempty"`
	StdinOnce       bool                `json:"StdinOnce,omitempty" yaml:"StdinOnce,omitempty"`
	Env             []string            `json:"Env,omitempty" yaml:"Env,omitempty"`
	Cmd             []string            `json:"Cmd" yaml:"Cmd"`
	DNS             []string            `json:"Dns,omitempty" yaml:"Dns,omitempty"` // For Docker API v1.9 and below only
	Image           string              `json:"Image,omitempty" yaml:"Image,omitempty"`
	Volumes         map[string]struct{} `json:"Volumes,omitempty" yaml:"Volumes,omitempty"`
	VolumeDriver    string              `json:"VolumeDriver,omitempty" yaml:"VolumeDriver,omitempty"`
	VolumesFrom     string              `json:"VolumesFrom,omitempty" yaml:"VolumesFrom,omitempty"`
	WorkingDir      string              `json:"WorkingDir,omitempty" yaml:"WorkingDir,omitempty"`
	MacAddress      string              `json:"MacAddress,omitempty" yaml:"MacAddress,omitempty"`
	Entrypoint      []string            `json:"Entrypoint" yaml:"Entrypoint"`
	NetworkDisabled bool                `json:"NetworkDisabled,omitempty" yaml:"NetworkDisabled,omitempty"`
	SecurityOpts    []string            `json:"SecurityOpts,omitempty" yaml:"SecurityOpts,omitempty"`
	OnBuild         []string            `json:"OnBuild,omitempty" yaml:"OnBuild,omitempty"`
	Mounts          []Mount             `json:"Mounts,omitempty" yaml:"Mounts,omitempty"`
	Labels          map[string]string   `json:"Labels,omitempty" yaml:"Labels,omitempty"`
}

// Image is the type representing a docker image and its various properties
type Image struct {
	ID              string    `json:"Id" yaml:"Id"`
	Parent          string    `json:"Parent,omitempty" yaml:"Parent,omitempty"`
	Comment         string    `json:"Comment,omitempty" yaml:"Comment,omitempty"`
	Created         time.Time `json:"Created,omitempty" yaml:"Created,omitempty"`
	Container       string    `json:"Container,omitempty" yaml:"Container,omitempty"`
	ContainerConfig Config    `json:"ContainerConfig,omitempty" yaml:"ContainerConfig,omitempty"`
	DockerVersion   string    `json:"DockerVersion,omitempty" yaml:"DockerVersion,omitempty"`
	Author          string    `json:"Author,omitempty" yaml:"Author,omitempty"`
	Config          *Config   `json:"Config,omitempty" yaml:"Config,omitempty"`
	Architecture    string    `json:"Architecture,omitempty" yaml:"Architecture,omitempty"`
	Size            int64     `json:"Size,omitempty" yaml:"Size,omitempty"`
	VirtualSize     int64     `json:"VirtualSize,omitempty" yaml:"VirtualSize,omitempty"`
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
	localAddrString  = "localhost:9000"
	unixDockerSocket = "unix:///var/run/docker.sock"

)
func main(){



	fmt.Println("Starting multi-tenancy proxy")

	// Setup localListener (type net.Listener)
	localListener, err := net.Listen("tcp", localAddrString)

	if err != nil {
		log.Fatalf("net.Listen failed: %v", err)
	}

	for {
		// Setup localConn (type net.Conn)
		localConn, err := localListener.Accept()
		if err != nil {
			log.Fatalf("listen.Accept failed: %v", err)
		}
		go forward(localConn)
	}




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
