package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/discovery"
	clientset "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	"github.com/vmware-tanzu/velero/pkg/podexec"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type clusterConnection struct {
	host               string
	veleroClient       clientset.Interface
	discoveryHelper    discovery.Helper
	dynamicFactory     client.DynamicFactory
	kubeClient         kubernetes.Interface
	podCommandExecutor podexec.PodCommandExecutor
}

func newClusterConnection(ctx context.Context, log *logrus.Logger, kubeAccess *kubeAccess) (*clusterConnection, error) {
	cxn := clusterConnection{
		host: remoteHost(kubeAccess),
	}

	veleroClient, err := kubeAccess.Client()
	if err != nil {
		return nil, fmt.Errorf("create velero client: %w", err)
	}
	cxn.veleroClient = veleroClient
	if err != nil {
		return nil, err
	}

	discoveryClient := veleroClient.Discovery()
	discoveryHelper, err := discovery.NewHelper(discoveryClient, log)
	if err != nil {
		return nil, fmt.Errorf("create discovery helper: %w", err)
	}
	cxn.discoveryHelper = discoveryHelper

	dynamicClient, err := kubeAccess.DynamicClient()
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}
	dynamicFactory := client.NewDynamicFactory(dynamicClient)
	cxn.dynamicFactory = dynamicFactory

	kubeClientConfig, err := kubeAccess.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("create kubeclient config: %w", err)
	}
	kubeClient, err := kubeAccess.KubeClient()
	if err != nil {
		return nil, fmt.Errorf("create kubeclient: %w", err)
	}
	cxn.kubeClient = kubeClient

	podCommandExecutor := podexec.NewPodCommandExecutor(
		kubeClientConfig,
		kubeClient.CoreV1().RESTClient(),
	)
	cxn.podCommandExecutor = podCommandExecutor

	return &cxn, nil
}

func remoteHost(access *kubeAccess) string {
	if access.details.host != "" {
		return access.details.host
	}

	return "in-cluster"
}

func retrieveAuthDetails(ctx context.Context, auth *kubeAccess) error {
	if auth.store.secretName == "" || auth.store.secretNamespace == "" {
		return nil
	}

	client, err := localClientset()
	if err != nil {
		return err
	}

	secrets, err := client.
		CoreV1().
		Secrets(auth.store.secretNamespace).
		List(ctx, meta.ListOptions{})
	if err != nil {
		return err
	}

	for _, item := range secrets.Items {
		if item.Name == auth.store.secretName {
			auth.details.host = string(item.Data["host"])
			auth.details.saToken = string(item.Data["sa-token"])
			auth.details.kubeconfig = string(item.Data["kubeconfig"])
			auth.details.httpsProxy = string(item.Data["https_proxy"])
			return nil
		}
	}

	// No credentials for remote cluster were found in authentication store.
	return nil
}

func localConfig() (*rest.Config, error) {
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	clientConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("retrieve local cluster kubeclient config: %w", err)
	}

	return clientConfig, nil
}

func localClientset() (*kubernetes.Clientset, error) {
	config, err := localConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create local cluster kubeclient: %w", err)
	}

	return client, nil
}

func setTransportProxy(config *rest.Config, proxy string) {
	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		transport := rt.(*http.Transport)
		proxyURL, _ := url.Parse(proxy)
		transport.Proxy = http.ProxyURL(proxyURL)
		return transport
	})
}
