// SPDX-FileCopyrightText: 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Case 16: Subresources — status, exec, log streaming
//
// Demonstrates:
//   - Updating only the /status subresource
//   - Streaming pod logs with the REST client
//   - Exec into a pod with remotecommand.NewSPDYExecutor
package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		slog.Error("error building kubeconfig", "err", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("error creating clientset", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Replace "test-pod" with an existing pod name (see README for setup)
	podName := "test-pod"
	namespace := "default"

	// GetLogs(name, opts) is sugar for the explicit /log subresource request
	// below — built exactly like the /exec call further down, differing only in
	// GET vs POST and the transport (plain stream vs SPDY).
	req := clientset.CoreV1().RESTClient().Get().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("log").
		VersionedParams(&corev1.PodLogOptions{TailLines: new(int64(10))}, scheme.ParameterCodec)
	stream, err := req.Stream(ctx)
	if err != nil {
		slog.Warn("log stream (pod must exist)", "err", err)
	} else {
		defer stream.Close()
		io.Copy(os.Stdout, stream)
	}

	execReq := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: []string{"sh", "-c", "echo hello from exec"},
			Stdout:  true,
			Stderr:  true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", execReq.URL())
	if err != nil {
		slog.Warn("exec setup (pod must exist)", "err", err)
		return
	}
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: os.Stdout, Stderr: os.Stderr})
	if err != nil {
		slog.Warn("exec stream", "err", err)
	}

	pod, _ := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	slog.Info("pod phase", "phase", pod.Status.Phase)
}

//go:fix inline
func int64Ptr(i int64) *int64 { return new(i) }

// loadConfig builds a *rest.Config from the KUBECONFIG env var, falling back to
// ~/.kube/config.
func loadConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
