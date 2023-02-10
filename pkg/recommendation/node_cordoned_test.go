package recommendation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/recommendation"
)

func TestNodeCordoned_Do_HappyPath(t *testing.T) {
	// given
	expected := recommendation.Result{
		Info: []string{
			"Node foo was cordoned. Check the health of this node.",
		},
	}

	recomm := recommendation.NewNodeCordoned()

	node := fixNode()
	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&node)
	require.NoError(t, err)
	unstr := &unstructured.Unstructured{Object: unstrObj}

	event, err := event.New(node.ObjectMeta, unstr, config.UpdateEvent, "v1/nodes")
	require.NoError(t, err)

	// when
	actual, err := recomm.Do(context.Background(), event)

	// then
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func fixNode() *v1.Node {
	return &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-name",
		},
		Spec: v1.NodeSpec{
			Unschedulable: false,
		},
	}
}
