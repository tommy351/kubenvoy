package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

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

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)

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

	RunSpecs(t, "e2e")
}

func portForwardPod(ctx context.Context, conf *rest.Config, pod *corev1.Pod, ports []string) (err error) {
	readyCh := make(chan struct{})
	serverURL, err := url.Parse(conf.Host)

	if err != nil {
		return err
	}

	serverURL.Path = fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", pod.Namespace, pod.Name)
	roundTripper, upgrader, err := spdy.RoundTripperFor(conf)

	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, serverURL)
	forwarder, err := portforward.New(dialer, ports, ctx.Done(), readyCh, ioutil.Discard, ioutil.Discard)

	go func() {
		err = forwarder.ForwardPorts()
	}()

	// Wait for ready
	for range readyCh {
	}

	return
}
