package recommendation

import (
	"context"
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/k8sutil"
)

const nodeCordonedName = "NodeCordoned"

// NodeCordoned adds recommendation when updated Nodes are unschedulable
type NodeCordoned struct{}

// NewNodeCordoned creates a new NodeCordoned instance.
func NewNodeCordoned() *NodeCordoned {
	return &NodeCordoned{}
}

// Do executes the recommendation checks.
func (f NodeCordoned) Do(_ context.Context, event event.Event) (Result, error) {
	if event.Kind != "Node" || event.Type != config.UpdateEvent {
		return Result{}, nil
	}

	unstrObj, ok := event.Object.(*unstructured.Unstructured)
	if !ok {
		return Result{}, fmt.Errorf("cannot convert %T into type %T", event.Object, unstrObj)
	}

	var node coreV1.Node
	err := k8sutil.TransformIntoTypedObject(unstrObj, &node)
	if err != nil {
		return Result{}, fmt.Errorf("while transforming object type %T into type: %T: %w", event.Object, node, err)
	}

	// Check if node is cordoned
	if !node.Spec.Unschedulable {
		return Result{}, nil
	} else {
		recommendationMsg := fmt.Sprintf("Node '%s' was cordoned. Check the health of this node.", node.Name)
		return Result{
			Info: []string{recommendationMsg},
		}, nil
	}
}

// Name returns the recommendation name.
func (f *NodeCordoned) Name() string {
	return nodeCordonedName
}
