package k8sexec

import (
	"context"
	"errors"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	exec "k8s.io/client-go/util/exec"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

var defaultConfigFlags = genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)

type K8SExecutor struct {
	restClient *restclient.RESTClient
	restConfig *restclient.Config
}

func NewK8SExecutor() (*K8SExecutor, error) {
	kubeConfigFlags := defaultConfigFlags

	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)

	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	config, err := f.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("could not create rest config: %w", err)
	}

	restClient, err := restclient.RESTClientFor(config)
	if err != nil {
		return nil, fmt.Errorf("could not create rest client: %w", err)
	}

	return &K8SExecutor{
		restConfig: config,
		restClient: restClient,
	}, nil
}

type WindowSize struct {
	Width  int
	Height int
}

func (k *K8SExecutor) RunOnPod(
	ctx context.Context,
	namespace,
	pod,
	container string,
	command []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	tty bool,
	windowSizeEvents chan WindowSize,
) error {
	var initialSize WindowSize
	select {
	case initialSize = <-windowSizeEvents:
	default:
		return errors.New("window size must have at least one element at start")
	}

	sq := &sizeQueue{
		resizeChan: make(chan remotecommand.TerminalSize, 1),
	}

	sq.resizeChan <- remotecommand.TerminalSize{
		Width:  uint16(initialSize.Width),
		Height: uint16(initialSize.Height),
	}

	go func() {
		for ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return
			case size := <-windowSizeEvents:
				sq.resizeChan <- remotecommand.TerminalSize{
					Width:  uint16(size.Width),
					Height: uint16(size.Height),
				}
			}
		}
	}()

	req := k.restClient.Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	method := "GET"
	ex, err := remotecommand.NewSPDYExecutor(k.restConfig, method, req.URL())
	if err != nil {
		return fmt.Errorf("could not create new executor: %w", err)
	}

	err = ex.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               tty,
		TerminalSizeQueue: sq,
	})
	if err != nil {
		ee := &exec.CodeExitError{}

		if errors.As(err, ee) {
			return ExitCodeError(ee.Code)
		}
		return fmt.Errorf("could not execute: %w", err)
	}

	return nil

}

type sizeQueue struct {
	resizeChan chan remotecommand.TerminalSize
}

func (s *sizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-s.resizeChan
	if !ok {
		return nil
	}
	return &size
}

type ExitCodeError int

func (e ExitCodeError) Error() string {
	return fmt.Sprintf("process exited with code %d", int(e))
}
