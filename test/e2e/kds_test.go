package e2e

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tommy351/kubenvoy/test/echo"
)

func decodeResponse(r *http.Response) (*echo.Response, error) {
	var res echo.Response
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&res); err != nil {
		return nil, err
	}

	return &res, nil
}

func mustDecodeResponse(r *http.Response) *echo.Response {
	data, err := decodeResponse(r)
	Expect(err).NotTo(HaveOccurred())
	return data
}

var _ = Describe("kds", func() {
	var (
		req *http.Request
		res *http.Response
		err error
	)

	BeforeEach(func() {
		req, err = http.NewRequest(http.MethodGet, "http://localhost:10000", nil)
		Expect(err).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		res, err = http.DefaultClient.Do(req)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("given host = single-port.echo", func() {
		BeforeEach(func() {
			req.Host = "single-port.echo"
		})

		It("should respond status 200", func() {
			Expect(res.StatusCode).To(Equal(http.StatusOK))
		})

		It("check response", func() {
			data := mustDecodeResponse(res)
			Expect(data.Method).To(Equal(http.MethodGet))
			Expect(data.Host).To(Equal(req.Host))
			Expect(data.URL).To(Equal("/"))
		})
	})

	Describe("given host = named-port.echo", func() {
		BeforeEach(func() {
			req.Host = "named-port.echo"
		})

		It("should respond status 200", func() {
			Expect(res.StatusCode).To(Equal(http.StatusOK))
		})

		It("check response", func() {
			data := mustDecodeResponse(res)
			Expect(data.Method).To(Equal(http.MethodGet))
			Expect(data.Host).To(Equal(req.Host))
			Expect(data.URL).To(Equal("/"))
		})
	})
})
