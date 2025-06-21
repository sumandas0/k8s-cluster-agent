package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/core/models"
)

func TestPodService_GetPod(t *testing.T) {
	testPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			NodeName: "test-node",
		},
	}

	fakeClient := fake.NewSimpleClientset(testPod)

	svc := NewPodService(fakeClient, slog.Default())

	pod, err := svc.GetPod(context.Background(), "default", "test-pod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pod.Name != "test-pod" {
		t.Errorf("expected pod name 'test-pod', got '%s'", pod.Name)
	}

	_, err = svc.GetPod(context.Background(), "default", "non-existent")
	if err != core.ErrPodNotFound {
		t.Errorf("expected ErrPodNotFound, got %v", err)
	}
}

func TestPodService_GetPodDescription(t *testing.T) {
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

	fakeClient := fake.NewSimpleClientset(testPod, testEvent)

	svc := NewPodService(fakeClient, slog.Default())

	description, err := svc.GetPodDescription(context.Background(), "default", "test-pod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if description.Name != "test-pod" {
		t.Errorf("expected name 'test-pod', got '%s'", description.Name)
	}
	if description.Namespace != "default" {
		t.Errorf("expected namespace 'default', got '%s'", description.Namespace)
	}
	if description.Node != "test-node" {
		t.Errorf("expected node 'test-node', got '%s'", description.Node)
	}

	if len(description.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(description.Labels))
	}
	if description.Labels["app"] != "test-app" {
		t.Errorf("expected label app='test-app', got '%s'", description.Labels["app"])
	}
	if len(description.Annotations) != 2 {
		t.Errorf("expected 2 annotations, got %d", len(description.Annotations))
	}

	if description.Status.Phase != "Running" {
		t.Errorf("expected phase 'Running', got '%s'", description.Status.Phase)
	}
	if description.Status.PodIP != "10.244.1.5" {
		t.Errorf("expected podIP '10.244.1.5', got '%s'", description.Status.PodIP)
	}

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

	if len(description.InitContainers) != 1 {
		t.Errorf("expected 1 init container, got %d", len(description.InitContainers))
	}
	initContainer := description.InitContainers[0]
	if initContainer.Name != "init-container" {
		t.Errorf("expected init container name 'init-container', got '%s'", initContainer.Name)
	}

	if len(description.Volumes) != 2 {
		t.Errorf("expected 2 volumes, got %d", len(description.Volumes))
	}

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

	if description.QOSClass != "Burstable" {
		t.Errorf("expected QoS class 'Burstable', got '%s'", description.QOSClass)
	}
	if description.Priority == nil || *description.Priority != 100 {
		t.Errorf("expected priority 100, got %v", description.Priority)
	}

	if len(description.Tolerations) != 1 {
		t.Errorf("expected 1 toleration, got %d", len(description.Tolerations))
	}
	if len(description.NodeSelector) != 1 {
		t.Errorf("expected 1 node selector, got %d", len(description.NodeSelector))
	}
	if description.NodeSelector["zone"] != "us-west-1" {
		t.Errorf("expected node selector zone='us-west-1', got '%s'", description.NodeSelector["zone"])
	}

	if len(description.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(description.Conditions))
	}

	_, err = svc.GetPodDescription(context.Background(), "default", "non-existent")
	if err != core.ErrPodNotFound {
		t.Errorf("expected ErrPodNotFound, got %v", err)
	}
}

func TestGetPodFailureEvents(t *testing.T) {
	createTestEvent := func(reason, message, eventType string, count int32, firstTime, lastTime time.Time) v1.Event {
		return v1.Event{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Event",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-event-%s", reason),
				Namespace: "test-namespace",
			},
			InvolvedObject: v1.ObjectReference{
				Kind:      "Pod",
				Name:      "test-pod",
				Namespace: "test-namespace",
			},
			Reason:         reason,
			Message:        message,
			Type:           eventType,
			Count:          count,
			FirstTimestamp: metav1.Time{Time: firstTime},
			LastTimestamp:  metav1.Time{Time: lastTime},
			Source: v1.EventSource{
				Component: "kubelet",
				Host:      "test-node",
			},
		}
	}

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)
	fiveMinutesAgo := now.Add(-5 * time.Minute)

	tests := []struct {
		name          string
		namespace     string
		podName       string
		pod           *v1.Pod
		events        *v1.EventList
		expectedError error
		validateFunc  func(t *testing.T, result *models.PodFailureEvents)
	}{
		{
			name:      "successful with critical failure events",
			namespace: "test-namespace",
			podName:   "test-pod",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-namespace",
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:         "test-container",
							RestartCount: 5,
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason:  "CrashLoopBackOff",
									Message: "Back-off restarting failed container",
								},
							},
						},
					},
				},
			},
			events: &v1.EventList{
				Items: []v1.Event{
					createTestEvent("CrashLoopBackOff", "Back-off restarting failed container", "Warning", 10, twoHoursAgo, fiveMinutesAgo),
					createTestEvent("FailedScheduling", "0/3 nodes are available: insufficient memory", "Warning", 15, oneHourAgo, now),
					createTestEvent("Pulled", "Successfully pulled image", "Normal", 1, oneHourAgo, oneHourAgo),
				},
			},
			validateFunc: func(t *testing.T, result *models.PodFailureEvents) {
				assert.Equal(t, "test-pod", result.PodName)
				assert.Equal(t, "test-namespace", result.Namespace)
				assert.Equal(t, 3, result.TotalEvents)
				assert.Equal(t, 2, len(result.FailureEvents))
				assert.Equal(t, 2, result.CriticalEvents)
				assert.Equal(t, 0, result.WarningEvents)
				assert.NotNil(t, result.MostRecentIssue)
				assert.Equal(t, "FailedScheduling", result.MostRecentIssue.Reason)

				assert.Equal(t, 2, len(result.EventCategories))
				assert.Equal(t, 1, result.EventCategories[models.FailureEventCategoryCrash])
				assert.Equal(t, 1, result.EventCategories[models.FailureEventCategoryScheduling])

				assert.Greater(t, len(result.OngoingIssues), 0)
			},
		},
		{
			name:      "pod with image pull failures",
			namespace: "test-namespace",
			podName:   "test-pod",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-namespace",
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			},
			events: &v1.EventList{
				Items: []v1.Event{
					createTestEvent("ImagePullBackOff", "Back-off pulling image \"invalid:latest\"", "Warning", 20, twoHoursAgo, fiveMinutesAgo),
					createTestEvent("ErrImagePull", "Failed to pull image \"invalid:latest\": rpc error", "Warning", 5, oneHourAgo, oneHourAgo),
				},
			},
			validateFunc: func(t *testing.T, result *models.PodFailureEvents) {
				assert.Equal(t, 2, len(result.FailureEvents))
				assert.Equal(t, 2, result.CriticalEvents)

				assert.Equal(t, 1, len(result.EventCategories))
				assert.Equal(t, 2, result.EventCategories[models.FailureEventCategoryImagePull])

				imagePullEvent := result.FailureEvents[0]
				assert.True(t, imagePullEvent.IsRecurring)
				assert.NotEmpty(t, imagePullEvent.RecurrenceRate)
				assert.NotEmpty(t, imagePullEvent.TimeSinceFirst)
			},
		},
		{
			name:      "pod with no failure events",
			namespace: "test-namespace",
			podName:   "test-pod",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-namespace",
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
			events: &v1.EventList{
				Items: []v1.Event{
					createTestEvent("Pulled", "Successfully pulled image", "Normal", 1, oneHourAgo, oneHourAgo),
					createTestEvent("Created", "Created container", "Normal", 1, oneHourAgo, oneHourAgo),
					createTestEvent("Started", "Started container", "Normal", 1, oneHourAgo, oneHourAgo),
				},
			},
			validateFunc: func(t *testing.T, result *models.PodFailureEvents) {
				assert.Equal(t, 3, result.TotalEvents)
				assert.Equal(t, 0, len(result.FailureEvents))
				assert.Equal(t, 0, result.CriticalEvents)
				assert.Equal(t, 0, result.WarningEvents)
				assert.Nil(t, result.MostRecentIssue)
				assert.Empty(t, result.EventCategories)
			},
		},
		{
			name:          "pod not found",
			namespace:     "test-namespace",
			podName:       "non-existent-pod",
			pod:           nil,
			events:        nil,
			expectedError: core.ErrPodNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objects []runtime.Object
			if tt.pod != nil {
				objects = append(objects, tt.pod)
			}
			if tt.events != nil {
				for i := range tt.events.Items {
					objects = append(objects, &tt.events.Items[i])
				}
			}
			fakeClient := fake.NewSimpleClientset(objects...)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			svc := NewPodService(fakeClient, logger)

			result, err := svc.GetPodFailureEvents(context.Background(), tt.namespace, tt.podName)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateFunc != nil {
					tt.validateFunc(t, result)
				}
			}
		})
	}
}

func TestAnalyzeFailureEvents(t *testing.T) {
	svc := &podService{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	now := time.Now()

	events := []models.EventInfo{
		{
			Type:           "Warning",
			Reason:         "OOMKilled",
			Message:        "Container was OOMKilled",
			FirstTimestamp: metav1.Time{Time: now.Add(-30 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: now.Add(-5 * time.Minute)},
			Count:          5,
		},
		{
			Type:           "Normal",
			Reason:         "BackOff",
			Message:        "Back-off restarting container",
			FirstTimestamp: metav1.Time{Time: now.Add(-1 * time.Hour)},
			LastTimestamp:  metav1.Time{Time: now.Add(-10 * time.Minute)},
			Count:          10},
	}

	pod := &v1.Pod{
		Status: v1.PodStatus{
			QOSClass: v1.PodQOSBurstable,
		},
	}

	results := svc.analyzeFailureEvents(events, pod)

	assert.Equal(t, 2, len(results))

	assert.Equal(t, "OOMKilled", results[0].Reason)
	assert.Equal(t, models.FailureEventCategoryResource, results[0].Category)
	assert.Equal(t, "critical", results[0].Severity)
	assert.True(t, results[0].IsRecurring)

	assert.Equal(t, "BackOff", results[1].Reason)
	assert.Equal(t, models.FailureEventCategoryCrash, results[1].Category)
}
