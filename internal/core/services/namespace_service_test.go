package services

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/sumandas0/k8s-cluster-agent/internal/config"
	"github.com/sumandas0/k8s-cluster-agent/internal/core/models"
)

func TestNamespaceService_GetNamespaceErrors(t *testing.T) {
	tests := []struct {
		name                string
		namespace           string
		restartThreshold    int
		pods                []runtime.Object
		events              []runtime.Object
		expectedTotalPods   int
		expectedProblematic int
		expectedCritical    int
		expectedWarning     int
		expectedError       bool
	}{
		{
			name:              "empty namespace",
			namespace:         "empty-ns",
			restartThreshold:  5,
			pods:              []runtime.Object{},
			expectedTotalPods: 0,
		},
		{
			name:             "healthy pods only",
			namespace:        "test-ns",
			restartThreshold: 5,
			pods: []runtime.Object{
				createPod("test-ns", "healthy-pod-1", "deployment", "Running", 0, 0),
				createPod("test-ns", "healthy-pod-2", "statefulset", "Running", 0, 2),
			},
			expectedTotalPods:   2,
			expectedProblematic: 0,
		},
		{
			name:             "pod with high restarts",
			namespace:        "test-ns",
			restartThreshold: 5,
			pods: []runtime.Object{
				createPod("test-ns", "restart-pod", "deployment", "Running", 0, 10),
			},
			expectedTotalPods:   1,
			expectedProblematic: 1,
			expectedCritical:    1,
		},
		{
			name:             "pending pod",
			namespace:        "test-ns",
			restartThreshold: 5,
			pods: []runtime.Object{
				createPendingPod("test-ns", "pending-pod", "deployment", time.Now().Add(-10*time.Minute)),
			},
			expectedTotalPods:   1,
			expectedProblematic: 1,
			expectedCritical:    1,
		},
		{
			name:             "crashloopbackoff pod",
			namespace:        "test-ns",
			restartThreshold: 5,
			pods: []runtime.Object{
				createCrashLoopPod("test-ns", "crash-pod", "deployment"),
			},
			expectedTotalPods:   1,
			expectedProblematic: 1,
			expectedCritical:    1,
		},
		{
			name:             "image pull error",
			namespace:        "test-ns",
			restartThreshold: 5,
			pods: []runtime.Object{
				createImagePullErrorPod("test-ns", "image-error-pod", "statefulset"),
			},
			expectedTotalPods:   1,
			expectedProblematic: 1,
			expectedCritical:    2,
		},
		{
			name:             "mixed healthy and problematic",
			namespace:        "test-ns",
			restartThreshold: 3,
			pods: []runtime.Object{
				createPod("test-ns", "healthy-1", "deployment", "Running", 0, 0),
				createPod("test-ns", "restart-pod", "deployment", "Running", 0, 5),
				createCrashLoopPod("test-ns", "crash-pod", "statefulset"),
				createPod("test-ns", "job-pod", "job", "Running", 0, 10),
			},
			expectedTotalPods:   3,
			expectedProblematic: 2,
			expectedCritical:    3,
		},
		{
			name:             "configurable restart threshold",
			namespace:        "test-ns",
			restartThreshold: 2,
			pods: []runtime.Object{
				createPod("test-ns", "pod-1", "deployment", "Running", 0, 3),
				createPod("test-ns", "pod-2", "deployment", "Running", 0, 1),
			},
			expectedTotalPods:   2,
			expectedProblematic: 1,
			expectedCritical:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(append(tt.pods, tt.events...)...)

			cfg := &config.Config{
				PodRestartThreshold: tt.restartThreshold,
			}

			logger := slog.Default()
			service := NewNamespaceService(fakeClient, cfg, logger)

			ctx := context.Background()
			report, err := service.GetNamespaceErrors(ctx, tt.namespace)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, report)

			assert.Equal(t, tt.namespace, report.Namespace)
			assert.Equal(t, tt.restartThreshold, report.RestartThresholdUsed)
			assert.Equal(t, tt.expectedTotalPods, report.TotalPodsAnalyzed)
			assert.Equal(t, tt.expectedProblematic, report.ProblematicPodsCount)
			assert.Equal(t, tt.expectedCritical, report.CriticalIssuesCount)
			assert.Equal(t, tt.expectedWarning, report.WarningIssuesCount)
		})
	}
}

func TestNamespaceService_filterPodsByOwner(t *testing.T) {
	tests := []struct {
		name     string
		pods     []v1.Pod
		expected int
	}{
		{
			name:     "empty list",
			pods:     []v1.Pod{},
			expected: 0,
		},
		{
			name: "deployment pods only",
			pods: []v1.Pod{
				*createPod("ns", "pod1", "deployment", "Running", 0, 0),
				*createPod("ns", "pod2", "deployment", "Running", 0, 0),
			},
			expected: 2,
		},
		{
			name: "mixed owner types",
			pods: []v1.Pod{
				*createPod("ns", "deploy-pod", "deployment", "Running", 0, 0),
				*createPod("ns", "ss-pod", "statefulset", "Running", 0, 0),
				*createPod("ns", "job-pod", "job", "Running", 0, 0),
				*createPod("ns", "daemonset-pod", "daemonset", "Running", 0, 0),
			},
			expected: 2,
		},
		{
			name: "no owner references",
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "orphan-pod",
						Namespace: "ns",
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{PodRestartThreshold: 5}
			logger := slog.Default()
			service := &namespaceService{
				k8sClient:           fake.NewSimpleClientset(),
				logger:              logger,
				podRestartThreshold: cfg.PodRestartThreshold,
			}

			filtered := service.filterPodsByOwner(tt.pods)
			assert.Len(t, filtered, tt.expected)
		})
	}
}

func TestNamespaceService_analyzePod(t *testing.T) {
	tests := []struct {
		name           string
		pod            *v1.Pod
		expectedIssues int
		expectedTypes  []models.PodIssueType
	}{
		{
			name:           "healthy pod",
			pod:            createPod("ns", "healthy", "deployment", "Running", 0, 0),
			expectedIssues: 0,
		},
		{
			name:           "high restart count",
			pod:            createPod("ns", "restart", "deployment", "Running", 0, 10),
			expectedIssues: 1,
			expectedTypes:  []models.PodIssueType{models.PodIssueHighRestarts},
		},
		{
			name:           "crashloop pod",
			pod:            createCrashLoopPod("ns", "crash", "deployment"),
			expectedIssues: 1,
			expectedTypes:  []models.PodIssueType{models.PodIssueCrashLoop},
		},
		{
			name:           "multiple issues",
			pod:            createMultiIssuePod("ns", "multi", "deployment"),
			expectedIssues: 2,
			expectedTypes:  []models.PodIssueType{models.PodIssueHighRestarts, models.PodIssueCrashLoop},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{PodRestartThreshold: 5}
			logger := slog.Default()
			fakeClient := fake.NewSimpleClientset()

			fakeClient.PrependReactor("list", "events", func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, &v1.EventList{Items: []v1.Event{}}, nil
			})

			service := &namespaceService{
				k8sClient:           fakeClient,
				logger:              logger,
				podRestartThreshold: cfg.PodRestartThreshold,
			}

			ctx := context.Background()
			result := service.analyzePod(ctx, tt.pod)

			assert.Equal(t, tt.pod.Name, result.Name)
			assert.Len(t, result.Issues, tt.expectedIssues)

			for i, expectedType := range tt.expectedTypes {
				assert.Equal(t, expectedType, result.Issues[i].Type)
			}
		})
	}
}

func createPod(namespace, name, ownerKind, phase string, _, restartCount int32) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Hour)),
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "main"},
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodPhase(phase),
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:         "main",
					RestartCount: restartCount,
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
				},
			},
		},
	}

	switch ownerKind {
	case "deployment":
		pod.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "ReplicaSet",
				Name: name + "-abc123",
			},
		}
	case "statefulset":
		pod.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "StatefulSet",
				Name: name + "-ss",
			},
		}
	case "job":
		pod.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "Job",
				Name: name + "-job",
			},
		}
	case "daemonset":
		pod.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "DaemonSet",
				Name: name + "-ds",
			},
		}
	}

	return pod
}

func createPendingPod(namespace, name, ownerKind string, creationTime time.Time) *v1.Pod {
	pod := createPod(namespace, name, ownerKind, "Pending", 0, 0)
	pod.CreationTimestamp = metav1.NewTime(creationTime)
	pod.Status.Conditions = []v1.PodCondition{
		{
			Type:    v1.PodScheduled,
			Status:  v1.ConditionFalse,
			Message: "0/3 nodes are available: insufficient memory.",
		},
	}
	return pod
}

func createCrashLoopPod(namespace, name, ownerKind string) *v1.Pod {
	pod := createPod(namespace, name, ownerKind, "Running", 1, 5)
	pod.Status.ContainerStatuses[0].State = v1.ContainerState{
		Waiting: &v1.ContainerStateWaiting{
			Reason:  "CrashLoopBackOff",
			Message: "Back-off restarting failed container",
		},
	}
	return pod
}

func createImagePullErrorPod(namespace, name, ownerKind string) *v1.Pod {
	pod := createPod(namespace, name, ownerKind, "Pending", 0, 0)
	pod.Status.ContainerStatuses[0].State = v1.ContainerState{
		Waiting: &v1.ContainerStateWaiting{
			Reason:  "ImagePullBackOff",
			Message: "Back-off pulling image \"nonexistent:latest\"",
		},
	}
	return pod
}

func createMultiIssuePod(namespace, name, ownerKind string) *v1.Pod {
	pod := createCrashLoopPod(namespace, name, ownerKind)
	pod.Status.ContainerStatuses[0].RestartCount = 10
	return pod
}
