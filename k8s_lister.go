package main

import (
	"context"

	"github.com/draganm/scotty/k8sutil"
)

type k8sLister struct {
	ctx context.Context
	kc  *k8sutil.Client
}

func (kl *k8sLister) ListNamespaces() ([]string, error) {
	return kl.kc.ListNamespaces(kl.ctx)
}

func (kl *k8sLister) ListPodsInNamespace(namespace string) ([]string, error) {
	return kl.kc.ListPods(kl.ctx, namespace)
}

func (kl *k8sLister) ListContainersInPod(namespace, pod string) ([]string, error) {
	return kl.kc.ListContainers(kl.ctx, namespace, pod)
}
