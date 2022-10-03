package controllers

import (
	"context"

	kcpapis "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	"github.com/kcp-dev/logicalcluster/v2"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ResourceReconciler reconciles a SampleSvc object
type ResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apis.kcp.dev,resources=apibindings,verbs=get;list;watch
//+kubebuilder:rbac:groups=apis.kcp.dev,resources=apibindings/status,verbs=get

// Reconcile reads that state of the cluster for a ToolchainCluster object and makes changes based on the state read
// and what is in the ToolchainCluster.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger = logger.WithValues("clusterName", req.ClusterName)

	var apiBindings kcpapis.APIBindingList
	if err := r.List(ctx, &apiBindings); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Resource controller listed all APIBindings across all workspaces", "count", len(apiBindings.Items))

	// Add the logical cluster to the context
	ctx = logicalcluster.WithCluster(ctx, logicalcluster.New(req.ClusterName))

	cmName := "test"
	cmNs := "default"
	cm := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: cmName, Namespace: cmNs}, cm)
	if err == nil || !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	cm = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: cmNs,
		},
		Data: map[string]string{
			"video_game": "Tomb Raider",
		},
	}

	err = r.Client.Create(ctx, cm)
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kcpapis.APIBinding{}).
		Complete(r)
}
