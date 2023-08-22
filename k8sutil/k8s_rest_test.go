package k8sutil_test

import (
	"context"
	"testing"

	"github.com/draganm/scotty/k8sutil"
	"github.com/stretchr/testify/require"
)

func XTestListNamespaces(t *testing.T) {
	require := require.New(t)

	c, err := k8sutil.NewClient()
	require.NoError(err)

	ns, err := c.ListNamespaces(context.Background())
	require.NoError(err)

	require.Equal(ns, nil)

}
