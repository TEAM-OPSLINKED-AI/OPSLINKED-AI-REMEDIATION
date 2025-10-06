package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

func NewK8sClient() (*kubernetes.Clientset, error) {
	// In-cluster configuration 시도
	config, err := rest.InClusterConfig()
	if err!= nil {
		// In-cluster 설정 실패 시, out-of-cluster (kubeconfig) 설정으로 fallback
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			home, err := os.UserHomeDir()
			if err!= nil {
				return nil, err
			}
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err!= nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err!= nil {
		return nil, err
	}
	return clientset, nil
}