package main

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	clientset "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// kubeAccess meets the Velero factory.go/factory interface.
type kubeAccess struct {
	store struct {
		secretName      string
		secretNamespace string
	}
	details struct {
		saToken    string
		host       string
		kubeconfig string
		httpsProxy string
		context    string
	}
}

func newKubeAccess(ctx context.Context, log *logrus.Logger, secretName, secretNamespace string) (*kubeAccess, error) {
	cxn := kubeAccess{
		store: struct {
			secretName, secretNamespace string
		}{
			secretName, secretNamespace,
		},
	}

	if err := retrieveAuthDetails(ctx, &cxn); err != nil {
		return nil, err
	}

	return &cxn, nil
}

// ClientConfig returns a config able to access a remote or local cluster.
func (kCxn kubeAccess) ClientConfig() (*rest.Config, error) {
	// Passing in a kubeconfig assumes TLS insecure is false. TLS information
	// must be provided in kubeconfig.
	if kCxn.details.kubeconfig != "" {
		fmt.Println("=== found remote kubeconfig ====")
		config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kCxn.details.kubeconfig))
		if err != nil {
			return nil, err
		}

		if kCxn.details.httpsProxy != "" {
			setTransportProxy(config, kCxn.details.httpsProxy)
		}

		return config, nil
	}

	// Passing in the SA token assumes TLS insecure is true. Only used if
	// kubeconfig has not been provided.
	if kCxn.details.saToken != "" && kCxn.details.host != "" {
		fmt.Println("=== found remote saToken ====")
		config := rest.Config{
			Host:            kCxn.details.host,
			BearerToken:     kCxn.details.saToken,
			TLSClientConfig: rest.TLSClientConfig{Insecure: true},
			Burst:           1000,
			QPS:             100,
		}

		if kCxn.details.httpsProxy != "" {
			setTransportProxy(&config, kCxn.details.httpsProxy)
		}

		return &config, nil
	}

	fmt.Println("=== no remote config found, using local cluster ====")
	// No remote cluster credentials were provided. Use local cluster config.
	return localConfig()
}

// Client retrieves a Velero clientset.
func (kCxn kubeAccess) Client() (clientset.Interface, error) {
	clientConfig, err := kCxn.ClientConfig()
	if err != nil {
		return nil, err
	}

	veleroClient, err := clientset.NewForConfig(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("create Velero client: %w", err)
	}
	return veleroClient, nil
}

// DynamicClient is able to work with unstructured objects on a Kubernetes cluster.
func (kCxn kubeAccess) DynamicClient() (dynamic.Interface, error) {
	clientConfig, err := kCxn.ClientConfig()
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}
	return dynamicClient, nil
}

// KubeClient creates a Kubeclient that is very close to the Velero Client.
func (kCxn kubeAccess) KubeClient() (kubernetes.Interface, error) {
	clientConfig, err := kCxn.ClientConfig()
	if err != nil {
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}
	return kubeClient, nil
}

// Satisfy Velero's factory.go/factory interface. Not yet needed for our purpose.
func (kCxn kubeAccess) BindFlags(flags *pflag.FlagSet) {
}

// Satisfy Velero's factory.go/factory interface. Not yet needed for our purpose.
func (kCxn kubeAccess) SetBasename(string) {

}

// Satisfy Velero's factory.go/factory interface. Not yet needed for our purpose.
func (kCxn kubeAccess) SetClientQPS(float32) {

}

// Satisfy Velero's factory.go/factory interface. Not yet needed for our purpose.
func (kCxn kubeAccess) SetClientBurst(int) {

}

// Satisfy Velero's factory.go/factory interface, but not needed.
func (kCxn kubeAccess) KubebuilderClient() (client.Client, error) {
	return nil, nil
}

// Satisfy Velero's factory.go/factory interface. Not yet needed for our purpose.
func (kCxn kubeAccess) Namespace() string {
	return ""
}
