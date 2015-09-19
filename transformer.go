package main

import (
	"net/http"
	"regexp"
)

type MultiTenancyTransformer interface {
	transformRequest(r *http.Request)
	transformResponse(r *http.Response)
}

type Transformer struct {
	regexp      *regexp.Regexp
	transformer MultiTenancyTransformer
}

type Transformers struct {
	transformers []*Transformer
}

var DockerTransformers *Transformers

func init() {

	DockerTransformers = NewDefaultTransformers()

}

func NewDefaultTransformers() *Transformers {
	ts := &Transformers{}

	ts.AddTransformer(NewImageListTransformer())

	return ts
}

func (ts *Transformers) AddTransformer(t *Transformer) {
	ts.transformers = append(ts.transformers, t)
}
