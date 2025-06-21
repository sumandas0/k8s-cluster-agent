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

func TestPodService_GetPodDescription(t *testing.T) {
	// Create a comprehensive test pod with various fields
	testPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app":     "test-app",
				"version": "1.0",
			},
			Annotations: map[string]string{
				"annotation1": "value1",
				"annotation2": "value2",
			},
		},
		Spec: v1.PodSpec{
			NodeName:          "test-node",
			SchedulerName:     "default-scheduler",
			Priority:          &[]int32{100}[0],
			PriorityClassName: "high-priority",
			NodeSelector: map[string]string{
				"zone": "us-west-1",
			},
			Tolerations: []v1.Toleration{
				{
					Key:      "node.kubernetes.io/not-ready",
					Operator: v1.TolerationOpExists,
					Effect:   v1.TaintEffectNoExecute,
				},
			},
			Containers: []v1.Container{
				{
					Name:  "test-container",
					Image: "nginx:1.20",
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("100m"),
							v1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("200m"),
							v1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
					Env: []v1.EnvVar{
						{Name: "ENV_VAR1", Value: "value1"},
						{Name: "ENV_VAR2", Value: "value2"},
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "config-volume",
							MountPath: "/etc/config",
							ReadOnly:  true,
						},
					},
				},
			},
			InitContainers: []v1.Container{
				{
					Name:  "init-container",
					Image: "busybox:1.35",
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "config-volume",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "test-config",
							},
						},
					},
				},
				{
					Name: "empty-dir-volume",
					VolumeSource: v1.VolumeSource{
						EmptyDir: &v1.EmptyDirVolumeSource{},
					},
				},
			},
		},
		Status: v1.PodStatus{
			Phase:  v1.PodRunning,
			Reason: "Running",
			HostIP: "192.168.1.100",
			PodIP:  "10.244.1.5",
			PodIPs: []v1.PodIP{
				{IP: "10.244.1.5"},
			},
			QOSClass: v1.PodQOSBurstable,
			StartTime: &metav1.Time{
				Time: metav1.Now().Time,
			},
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:         "test-container",
					Image:        "nginx:1.20",
					ImageID:      "docker-pullable://nginx@sha256:abc123",
					Ready:        true,
					RestartCount: 0,
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{
							StartedAt: metav1.Now(),
						},
					},
				},
			},
			InitContainerStatuses: []v1.ContainerStatus{
				{
					Name:         "init-container",
					Image:        "busybox:1.35",
					ImageID:      "docker-pullable://busybox@sha256:def456",
					Ready:        true,
					RestartCount: 0,
					State: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							ExitCode: 0,
							Reason:   "Completed",
						},
					},
				},
			},
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	// Create test events
	testEvent := &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: "default",
		},
		InvolvedObject: v1.ObjectReference{
			Kind:      "Pod",
			Name:      "test-pod",
			Namespace: "default",
		},
		Type:           "Normal",
		Reason:         "Scheduled",
		Message:        "Successfully assigned pod to node",
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          1,
		Source: v1.EventSource{
			Component: "default-scheduler",
			Host:      "master-node",
		},
	}

	// Create fake client with test pod and event
	fakeClient := fake.NewSimpleClientset(testPod, testEvent)

	// Create service
	svc := NewPodService(fakeClient, slog.Default())

	// Test successful description
	description, err := svc.GetPodDescription(context.Background(), "default", "test-pod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Validate basic information
	if description.Name != "test-pod" {
		t.Errorf("expected name 'test-pod', got '%s'", description.Name)
	}
	if description.Namespace != "default" {
		t.Errorf("expected namespace 'default', got '%s'", description.Namespace)
	}
	if description.Node != "test-node" {
		t.Errorf("expected node 'test-node', got '%s'", description.Node)
	}

	// Validate labels and annotations
	if len(description.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(description.Labels))
	}
	if description.Labels["app"] != "test-app" {
		t.Errorf("expected label app='test-app', got '%s'", description.Labels["app"])
	}
	if len(description.Annotations) != 2 {
		t.Errorf("expected 2 annotations, got %d", len(description.Annotations))
	}

	// Validate status
	if description.Status.Phase != "Running" {
		t.Errorf("expected phase 'Running', got '%s'", description.Status.Phase)
	}
	if description.Status.PodIP != "10.244.1.5" {
		t.Errorf("expected podIP '10.244.1.5', got '%s'", description.Status.PodIP)
	}

	// Validate containers
	if len(description.Containers) != 1 {
		t.Errorf("expected 1 container, got %d", len(description.Containers))
	}
	container := description.Containers[0]
	if container.Name != "test-container" {
		t.Errorf("expected container name 'test-container', got '%s'", container.Name)
	}
	if container.Image != "nginx:1.20" {
		t.Errorf("expected container image 'nginx:1.20', got '%s'", container.Image)
	}
	if !container.Ready {
		t.Error("expected container to be ready")
	}

	// Validate init containers
	if len(description.InitContainers) != 1 {
		t.Errorf("expected 1 init container, got %d", len(description.InitContainers))
	}
	initContainer := description.InitContainers[0]
	if initContainer.Name != "init-container" {
		t.Errorf("expected init container name 'init-container', got '%s'", initContainer.Name)
	}

	// Validate volumes
	if len(description.Volumes) != 2 {
		t.Errorf("expected 2 volumes, got %d", len(description.Volumes))
	}

	// Find the ConfigMap volume
	var configMapVolume *v1.VolumeSource
	for _, vol := range description.Volumes {
		if vol.Name == "config-volume" {
			if vol.Type != "ConfigMap" {
				t.Errorf("expected volume type 'ConfigMap', got '%s'", vol.Type)
			}
			configMapVolume = &vol.Source
			break
		}
	}
	if configMapVolume == nil {
		t.Error("config-volume not found")
	}

	// Validate QoS and Priority
	if description.QOSClass != "Burstable" {
		t.Errorf("expected QoS class 'Burstable', got '%s'", description.QOSClass)
	}
	if description.Priority == nil || *description.Priority != 100 {
		t.Errorf("expected priority 100, got %v", description.Priority)
	}

	// Validate tolerations and node selector
	if len(description.Tolerations) != 1 {
		t.Errorf("expected 1 toleration, got %d", len(description.Tolerations))
	}
	if len(description.NodeSelector) != 1 {
		t.Errorf("expected 1 node selector, got %d", len(description.NodeSelector))
	}
	if description.NodeSelector["zone"] != "us-west-1" {
		t.Errorf("expected node selector zone='us-west-1', got '%s'", description.NodeSelector["zone"])
	}

	// Validate conditions
	if len(description.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(description.Conditions))
	}

	// Test pod not found
	_, err = svc.GetPodDescription(context.Background(), "default", "non-existent")
	if err != core.ErrPodNotFound {
		t.Errorf("expected ErrPodNotFound, got %v", err)
	}
}
