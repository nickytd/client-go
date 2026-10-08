// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Package v1 is the newer version of the Widget API.
//
// Compared to v1alpha1 it restructures the flat Size into a Dimensions struct
// (Width/Height) and adds a Material field. The conversion functions in the
// parent package translate between v1alpha1 and v1.
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersion is the GVK group/version this package registers.
var GroupVersion = schema.GroupVersion{Group: "example.com", Version: "v1"}

// Widget is the v1 custom resource.
type Widget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              WidgetSpec `json:"spec"`
}

// WidgetSpec is the v1 spec: Size is replaced by structured Dimensions, and a
// new Material field is added.
type WidgetSpec struct {
	Color      string     `json:"color"`
	Dimensions Dimensions `json:"dimensions"`
	Material   string     `json:"material"`
}

// Dimensions carries the width and height that replaced v1alpha1's flat Size.
type Dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// WidgetList is the list form of Widget.
type WidgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Widget `json:"items"`
}

// DeepCopyObject returns a copy as a runtime.Object. Dimensions is a value
// type, so a shallow struct copy is a complete copy for Widget.
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

// AddToScheme registers the v1 types with the given scheme.
func AddToScheme(scheme *runtime.Scheme) {
	scheme.AddKnownTypes(GroupVersion, &Widget{}, &WidgetList{})
	metav1.AddToGroupVersion(scheme, GroupVersion)
}
