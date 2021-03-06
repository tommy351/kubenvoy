package k8s

import (
	"os"
	"path/filepath"

	"github.com/ansel1/merry"
	homedir "github.com/mitchellh/go-homedir"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func LoadConfig() (*rest.Config, error) {
	conf, err := rest.InClusterConfig()

	if err == nil {
		return conf, nil
	}

	if err != rest.ErrNotInCluster {
		return nil, merry.Wrap(err)
	}

	path := os.Getenv("KUBECONFIG")

	if path == "" {
		home, err := homedir.Dir()

		if err != nil {
			return nil, merry.Wrap(err)
		}

		path = filepath.Join(home, ".kube", "config")
	}

	return clientcmd.BuildConfigFromFlags("", path)
}
