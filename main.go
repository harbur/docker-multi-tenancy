package main

import (
	"net/http"
	"errors"
	"io/ioutil"
	"log"
)

const userAgent = "docker-multi-tenancy"
const logResponse bool = true

var (
	// ErrInvalidEndpoint is returned when the endpoint is not a valid HTTP URL.
	ErrInvalidEndpoint = errors.New("invalid endpoint")

	ErrConnectionRefused = errors.New("Connection refused")
)



var (
	localAddrString  = ":9000"
	unixDockerSocket = "unix:///var/run/docker.sock"

)


func dockerRequestHandler(ts *Transformers) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var trans *Transformer
		log.Println("Recieved methos ", r.Method, " path", r.URL.Path)
		c, err := NewClient(unixDockerSocket)

		if err != nil {
			w.Write([]byte("Error"))
			return
		}

		for _, t := range ts.transformers{
			m := t.regexp.FindStringSubmatch(r.URL.Path)
			if m != nil {
				trans = t
				break

			}else{
				log.Printf("Transformer %v  did not math to path %v\n", t.regexp, r.URL.Path)
			}
		}

		if trans != nil {
			log.Println("Applying transformation to request")
			trans.transformer.transformRequest(r)
		}

		resp, err := c.do( r.Method,  r.URL.String(), r.Body)

		if trans != nil && err == nil {
			log.Println("Applying transformation to response")
			trans.transformer.transformResponse(resp)
		}

		if resp != nil {
			defer resp.Body.Close()
			content, _ := ioutil.ReadAll(resp.Body)
			w.Write(content)
		}
	}
	return http.HandlerFunc(fn)
}




func main(){

	log.Println("Starting multi-tenancy proxy listening at: ", localAddrString)


	http.ListenAndServe(localAddrString, dockerRequestHandler(DockerTransformers))

}

