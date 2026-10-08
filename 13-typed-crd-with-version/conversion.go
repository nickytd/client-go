// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/home-lab/client-go/13-typed-crd-with-version/api/v1"
	"github.com/home-lab/client-go/13-typed-crd-with-version/api/v1alpha1"
)

// defaultMaterial is assigned to v1 objects produced from a v1alpha1 source,
// which has no Material field to carry over.
const defaultMaterial = "unknown"

// RegisterConversions wires both directions of the v1alpha1 <-> v1 conversion
// into the scheme. Once registered, scheme.Convert (and any codec built from
// the scheme) can translate between the two versions automatically.
func RegisterConversions(scheme *runtime.Scheme) error {
	if err := scheme.AddConversionFunc((*v1alpha1.Widget)(nil), (*v1.Widget)(nil),
		func(a, b any, _ conversion.Scope) error {
			return convertV1alpha1ToV1(a.(*v1alpha1.Widget), b.(*v1.Widget))
		}); err != nil {
		return err
	}
	return scheme.AddConversionFunc((*v1.Widget)(nil), (*v1alpha1.Widget)(nil),
		func(a, b any, _ conversion.Scope) error {
			return convertV1ToV1alpha1(a.(*v1.Widget), b.(*v1alpha1.Widget))
		})
}

// convertV1alpha1ToV1 is the "up" conversion. The flat Size becomes a square
// (Width == Height == Size), and the new Material field is defaulted since the
// source has nothing to supply.
func convertV1alpha1ToV1(in *v1alpha1.Widget, out *v1.Widget) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec.Color = in.Spec.Color
	out.Spec.Dimensions = v1.Dimensions{Width: in.Spec.Size, Height: in.Spec.Size}
	out.Spec.Material = defaultMaterial
	return nil
}

// convertV1ToV1alpha1 is the "down" conversion. It is intentionally LOSSY:
// v1alpha1 has a single Size, so Height and Material cannot be represented and
// are dropped. We keep Width as the Size.
func convertV1ToV1alpha1(in *v1.Widget, out *v1alpha1.Widget) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec.Color = in.Spec.Color
	out.Spec.Size = in.Spec.Dimensions.Width
	// in.Spec.Dimensions.Height and in.Spec.Material are lost.
	return nil
}
