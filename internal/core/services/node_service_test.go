package services

import (
	"context"
	"log/slog"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
)

func TestNodeService_GetNodeUtilization_NoMetrics(t *testing.T) {
	testNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Status: v1.NodeStatus{
			Capacity: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("4"),
				v1.ResourceMemory: resource.MustParse("8Gi"),
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(testNode)

	svc := NewNodeService(fakeClient, nil, slog.Default())

	_, err := svc.GetNodeUtilization(context.Background(), "test-node")
	if err != core.ErrMetricsNotAvailable {
		t.Errorf("expected ErrMetricsNotAvailable, got %v", err)
	}

	_, err = svc.GetNodeUtilization(context.Background(), "non-existent")
	if err != core.ErrNodeNotFound {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		name     string
		usage    *resource.Quantity
		capacity *resource.Quantity
		want     float64
	}{
		{
			name:     "normal calculation",
			usage:    resource.NewQuantity(1000, resource.DecimalSI),
			capacity: resource.NewQuantity(4000, resource.DecimalSI),
			want:     25.0,
		},
		{
			name:     "zero capacity",
			usage:    resource.NewQuantity(1000, resource.DecimalSI),
			capacity: resource.NewQuantity(0, resource.DecimalSI),
			want:     0.0,
		},
		{
			name:     "zero usage",
			usage:    resource.NewQuantity(0, resource.DecimalSI),
			capacity: resource.NewQuantity(4000, resource.DecimalSI),
			want:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePercentage(tt.usage, tt.capacity)
			if got != tt.want {
				t.Errorf("calculatePercentage() = %v, want %v", got, tt.want)
			}
		})
	}
}
