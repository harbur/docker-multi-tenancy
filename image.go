package main

import (
	"log"
	"net/http"
	"regexp"
)

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

type imageTransformer struct{}

func NewImageListTransformer() *Transformer {
	t := &Transformer{}

	t.regexp = regexp.MustCompile("/v.*/images/json")

	t.transformer = &imageTransformer{}

	return t
}

func (c *imageTransformer) transformRequest(r *http.Request) {
	log.Println("Modifiy somehow the request")
}

func (c *imageTransformer) transformResponse(r *http.Response) {
	log.Println("Modifiy somehow the resopnse")
}
