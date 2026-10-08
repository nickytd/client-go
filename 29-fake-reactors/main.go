// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 29: Fake reactors & dynamic fake client
//
// Case 18 showed the fake clientset for happy-path tests. This case shows how
// to make the fake misbehave on purpose — PrependReactor injects errors or
// canned responses so you can test a controller's failure handling — and
// introduces the dynamic fake client for unstructured/CRD code.
//
// The substance lives in reactors_test.go; run it with `go test ./...`.
package main

import "fmt"

func main() {
	fmt.Println("This case is driven by tests. Run: go test ./...")
}
