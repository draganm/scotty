package k8sutil

import (
	"context"
	"fmt"
	"sort"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var defaultConfigFlags = genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)

type Client struct {
	restClient *restclient.RESTClient
	restConfig *restclient.Config
	clientset  *kubernetes.Clientset
	f          cmdutil.Factory
}

func NewClient() (*Client, error) {

	kubeConfigFlags := defaultConfigFlags

	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)

	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	config, err := f.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("could not create rest config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not get clientset: %w", err)
	}

	restClient, err := restclient.RESTClientFor(config)
	if err != nil {
		return nil, fmt.Errorf("could not create rest client: %w", err)
	}

	return &Client{
		restClient: restClient,
		restConfig: config,
		clientset:  clientset,
		f:          f,
	}, nil

}

func (c *Client) ListNamespaces(ctx context.Context) ([]string, error) {
	r := c.f.NewBuilder().
		Unstructured().
		// AllNamespaces(true).
		// RequestChunksOf(cmdutil.DefaultChunkSize).
		ResourceTypeOrNameArgs(true, "ns").
		ContinueOnError().
		// Latest().
		Flatten().
		// TransformRequests(o.transformRequests).
		Do()

	infos, err := r.Infos()
	if err != nil {
		return nil, fmt.Errorf("could not list namespaces: %w", err)
	}

	res := []string{}

	for _, i := range infos {
		res = append(res, i.Name)
	}

	sort.Strings(res)

	return res, nil

}

func (c *Client) ListPods(ctx context.Context, ns string) ([]string, error) {
	r := c.f.NewBuilder().
		Unstructured().
		NamespaceParam(ns).
		// RequestChunksOf(cmdutil.DefaultChunkSize).
		ResourceTypeOrNameArgs(true, "pod").
		ContinueOnError().
		// Latest().
		Flatten().
		// TransformRequests(o.transformRequests).
		Do()

	infos, err := r.Infos()
	if err != nil {
		return nil, fmt.Errorf("could not list namespaces: %w", err)
	}

	res := []string{}

	for _, i := range infos {
		res = append(res, i.Name)
	}

	sort.Strings(res)

	return res, nil

}

func (c *Client) ListContainers(ctx context.Context, ns, pod string) ([]string, error) {
	p, err := c.clientset.CoreV1().Pods(ns).Get(ctx, pod, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not find pod %s in ns %s: %w", pod, ns, err)
	}

	containers := []string{}

	for _, c := range p.Spec.Containers {
		containers = append(containers, c.Name)
	}

	sort.Strings(containers)

	return containers, nil
	// r := c.f.NewBuilder().
	// 	Unstructured().
	// 	NamespaceParam(ns).
	// 	// RequestChunksOf(cmdutil.DefaultChunkSize).
	// 	ResourceTypeOrNameArgs(false, "pod", pod).
	// 	ContinueOnError().
	// 	// Latest().
	// 	Flatten().
	// 	// TransformRequests(o.transformRequests).
	// 	Do()

	// infos, err := r.Infos()
	// if err != nil {
	// 	return nil, fmt.Errorf("could not list namespaces: %w", err)
	// }

	// if len(infos) != 1 {
	// 	return nil, fmt.Errorf("expected 1, got %d pod infos", len(infos))
	// }

	// res := []string{}

	// for _, i := range infos {
	// 	res = append(res, i.Name)
	// 	spew.Dump(i.Object)
	// }

	// sort.Strings(res)

	// return res, nil

}
