package main

import (
	"net/http"

	"github.com/tommy351/kubenvoy/test/echo"
)

func main() {
	handler := new(echo.Handler)

	if err := http.ListenAndServe(":80", handler); err != nil {
		panic(err)
	}
}
