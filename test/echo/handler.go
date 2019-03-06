package echo

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type Handler struct{}

func (*Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	res := Response{
		Method: r.Method,
		URL:    r.URL.String(),
		Host:   r.Host,
		Header: r.Header,
		Body:   body,
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	if err := encoder.Encode(&res); err != nil {
		panic(err)
	}
}
