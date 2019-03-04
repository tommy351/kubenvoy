package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tommy351/kubenvoy/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func portForwardPod(ctx context.Context, conf *rest.Config, pod *corev1.Pod, ports []string) (err error) {
	readyCh := make(chan struct{})
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", pod.Namespace, pod.Name)
	hostIP := strings.TrimPrefix(conf.Host, "https:/")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}
	roundTripper, upgrader, err := spdy.RoundTripperFor(conf)

	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)
	forwarder, err := portforward.New(dialer, ports, ctx.Done(), readyCh, ioutil.Discard, ioutil.Discard)

	go func() {
		err = forwarder.ForwardPorts()
	}()

	// Wait for ready
	for range readyCh {
	}

	return
}

var _ = Describe("kds", func() {
	It("Get", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		conf, err := k8s.LoadConfig()
		Expect(err).NotTo(HaveOccurred())

		client, err := kubernetes.NewForConfig(conf)
		Expect(err).NotTo(HaveOccurred())

		pods, err := client.CoreV1().Pods("default").List(metav1.ListOptions{
			LabelSelector: "app=kds",
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(portForwardPod(ctx, conf, &pods.Items[0], []string{"10000"})).NotTo(HaveOccurred())
	})
})
