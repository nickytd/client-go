// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 13 (versioned): Multi-version typed CRD with in-process conversion
//
// Builds on 13-typed-crd by registering TWO hand-written versions of the
// Widget API (v1alpha1 and v1) in one scheme, plus conversion functions
// between them. Unlike case 13 — which used codecs.WithoutConversion() — this
// case uses the scheme's converter to translate objects between versions
// entirely in-process, no API server required.
//
// Demonstrates:
//   - Registering multiple GroupVersions in a single scheme
//   - scheme.AddConversionFunc for both directions
//   - A lossy down-conversion (v1 -> v1alpha1 drops fields)
//   - scheme.Convert as the primitive codecs use on decode
package main

import (
	"log/slog"
	"os"

	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/home-lab/client-go/13-typed-crd-with-version/api/v1"
	"github.com/home-lab/client-go/13-typed-crd-with-version/api/v1alpha1"
)

func main() {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	v1.AddToScheme(scheme)
	if err := RegisterConversions(scheme); err != nil {
		slog.Error("error registering conversions", "err", err)
		os.Exit(1)
	}

	// Start with a v1alpha1 Widget: a flat Size of 5.
	src := &v1alpha1.Widget{Spec: v1alpha1.WidgetSpec{Color: "blue", Size: 5}}
	slog.Info("source v1alpha1",
		"color", src.Spec.Color, "size", src.Spec.Size)

	// Up-convert v1alpha1 -> v1. Size becomes a 5x5 square; Material defaults.
	up := &v1.Widget{}
	if err := scheme.Convert(src, up, nil); err != nil {
		slog.Error("up-conversion failed", "err", err)
		os.Exit(1)
	}
	slog.Info("converted to v1",
		"color", up.Spec.Color,
		"width", up.Spec.Dimensions.Width,
		"height", up.Spec.Dimensions.Height,
		"material", up.Spec.Material)

	// Mutate the v1 object to show a non-square, materialed widget, then
	// down-convert v1 -> v1alpha1. This is LOSSY: Height and Material vanish.
	up.Spec.Dimensions = v1.Dimensions{Width: 8, Height: 3}
	up.Spec.Material = "aluminium"
	slog.Info("mutated v1",
		"width", up.Spec.Dimensions.Width,
		"height", up.Spec.Dimensions.Height,
		"material", up.Spec.Material)

	down := &v1alpha1.Widget{}
	if err := scheme.Convert(up, down, nil); err != nil {
		slog.Error("down-conversion failed", "err", err)
		os.Exit(1)
	}
	slog.Info("converted back to v1alpha1 (lossy: height and material dropped)",
		"color", down.Spec.Color, "size", down.Spec.Size)
}
