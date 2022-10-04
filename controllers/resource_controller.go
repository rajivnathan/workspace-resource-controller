package controllers

import (
	"bytes"
	"context"
	"io"

	"github.com/go-logr/logr"
	"github.com/rajivnathan/workspace-resource-controller/templates"

	kcpapis "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	"github.com/kcp-dev/logicalcluster/v2"
	errs "github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// ResourceReconciler reconciles a SampleSvc object
type ResourceReconciler struct {
	client.Client
	Config            *rest.Config
	Scheme            *runtime.Scheme
	ResourceTemplates templates.TemplateData
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

	tmplBytes, err := templates.RenderResources(r.ResourceTemplates.Content, r.ResourceTemplates.Args)
	if err != nil {
		return ctrl.Result{}, err
	}

	// cmName := "test"
	// cmNs := "default"
	// cm := &corev1.ConfigMap{}
	// err = r.Client.Get(ctx, types.NamespacedName{Name: cmName, Namespace: cmNs}, cm)
	// if err == nil || !errors.IsNotFound(err) {
	// 	return ctrl.Result{}, err
	// }

	// cm = &corev1.ConfigMap{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      cmName,
	// 		Namespace: cmNs,
	// 	},
	// 	Data: map[string]string{
	// 		"video_game": "Tomb Raider",
	// 	},
	// }

	// err = r.Client.Create(ctx, cm)
	return ctrl.Result{}, r.createResourcesFromTemplate(logger, ctx, tmplBytes)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kcpapis.APIBinding{}).
		Complete(r)
}

func (r *ResourceReconciler) createResourcesFromTemplate(logger logr.Logger, ctx context.Context, templateData []byte) error {
	// decode template into objects
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(templateData), 100)
	var objsToProcess []runtime.RawExtension
	var err error
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			break
		}
		objsToProcess = append(objsToProcess, rawObj)
	}
	if err != io.EOF {
		return err
	}

	// create objects
	for _, rawObj := range objsToProcess {
		err := r.createObject(logger, ctx, rawObj)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *ResourceReconciler) createObject(logger logr.Logger, ctx context.Context, rawObj runtime.RawExtension) error {

	obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	if err != nil {
		return err
	}

	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

	dynamicCl, err := dynamic.NewForConfig(r.Config)
	if err != nil {
		return err
	}
	dc, err := discovery.NewDiscoveryClientForConfig(r.Config)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	// unstructuredObj.SetNamespace(p.namespace)
	// dri := dynamicCl.Resource(mapping.Resource).Namespace(p.namespace)
	dri := dynamicCl.Resource(mapping.Resource)

	logger.Info("Creating object", "name", unstructuredObj.GetName(), "namespace", unstructuredObj.GetNamespace())

	_, err = dri.Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		err = errs.Wrapf(err, "problem creating %+v", unstructuredObj)
	}
	return err
}
