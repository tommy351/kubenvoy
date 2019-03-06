package echo

import "net/http"

type Response struct {
	Method string
	URL    string
	Host   string
	Header http.Header
	Body   []byte
}
