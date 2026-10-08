// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Package v1alpha1 is the original (older) version of the Widget API.
//
// Its spec is intentionally simple: a flat Size integer. The v1 package
// restructures this into Dimensions and adds a Material field; the conversion
// functions in the parent package translate between the two.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersion is the GVK group/version this package registers.
var GroupVersion = schema.GroupVersion{Group: "example.com", Version: "v1alpha1"}

// Widget is the v1alpha1 custom resource.
type Widget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              WidgetSpec `json:"spec"`
}

// WidgetSpec is the v1alpha1 spec: a flat size.
type WidgetSpec struct {
	Color string `json:"color"`
	Size  int    `json:"size"`
}

// WidgetList is the list form of Widget.
type WidgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Widget `json:"items"`
}

// DeepCopyObject returns a copy as a runtime.Object. Shallow is fine here;
// real controller-gen output deep-copies nested fields (e.g. the Items slice).
func (w *Widget) DeepCopyObject() runtime.Object { out := *w; return &out }

// DeepCopyObject returns a copy as a runtime.Object.
func (l *WidgetList) DeepCopyObject() runtime.Object {
	out := *l
	if l.Items != nil {
		out.Items = make([]Widget, len(l.Items))
		copy(out.Items, l.Items)
	}
	return &out
}

// AddToScheme registers the v1alpha1 types with the given scheme.
func AddToScheme(scheme *runtime.Scheme) {
	scheme.AddKnownTypes(GroupVersion, &Widget{}, &WidgetList{})
	metav1.AddToGroupVersion(scheme, GroupVersion)
}
