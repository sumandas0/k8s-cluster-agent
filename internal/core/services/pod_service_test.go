package services

import (
	"context"
	"log/slog"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
)

func TestPodService_GetPod(t *testing.T) {
	// Create test pod
	testPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			NodeName: "test-node",
		},
	}

	// Create fake client with test pod
	fakeClient := fake.NewSimpleClientset(testPod)

	// Create service
	svc := NewPodService(fakeClient, slog.Default())

	// Test successful get
	pod, err := svc.GetPod(context.Background(), "default", "test-pod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pod.Name != "test-pod" {
		t.Errorf("expected pod name 'test-pod', got '%s'", pod.Name)
	}

	// Test pod not found
	_, err = svc.GetPod(context.Background(), "default", "non-existent")
	if err != core.ErrPodNotFound {
		t.Errorf("expected ErrPodNotFound, got %v", err)
	}
}
