// Package v1alpha1 contains the API schema for the ops.failover.dev/v1alpha1
// API group. It defines the FailoverApp custom resource.
//
// +kubebuilder:object:generate=true
// +groupName=ops.failover.dev
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is the group/version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "ops.failover.dev", Version: "v1alpha1"}

	// SchemeBuilder registers the API types with a runtime scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group/version to a scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
