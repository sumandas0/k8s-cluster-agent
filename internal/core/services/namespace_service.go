package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sumandas0/k8s-cluster-agent/internal/config"
	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/core/models"
)

type namespaceService struct {
	k8sClient           kubernetes.Interface
	logger              *slog.Logger
	podRestartThreshold int
}

func NewNamespaceService(k8sClient kubernetes.Interface, cfg *config.Config, logger *slog.Logger) core.NamespaceService {
	return &namespaceService{
		k8sClient:           k8sClient,
		logger:              logger,
		podRestartThreshold: cfg.PodRestartThreshold,
	}
}

func (s *namespaceService) GetNamespaceErrors(ctx context.Context, namespace string) (*models.NamespaceErrorReport, error) {
	s.logger.Debug("analyzing namespace for errors",
		"namespace", namespace,
		"restartThreshold", s.podRestartThreshold)

	report := &models.NamespaceErrorReport{
		Namespace:            namespace,
		AnalysisTime:         time.Now(),
		RestartThresholdUsed: s.podRestartThreshold,
		ProblematicPods:      []models.ProblematicPod{},
		Summary:              []models.NamespaceErrorSummary{},
	}

	pods, err := s.k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return report, nil
		}
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	filteredPods := s.filterPodsByOwner(pods.Items)
	report.TotalPodsAnalyzed = len(filteredPods)

	issueSummary := make(map[models.PodIssueType]*models.NamespaceErrorSummary)

	for i := range filteredPods {
		pod := &filteredPods[i]
		problematicPod := s.analyzePod(ctx, pod)

		if len(problematicPod.Issues) > 0 {
			report.ProblematicPods = append(report.ProblematicPods, *problematicPod)
			report.ProblematicPodsCount++

			for _, issue := range problematicPod.Issues {
				if summary, exists := issueSummary[issue.Type]; exists {
					summary.Count++
					summary.AffectedPods = append(summary.AffectedPods, pod.Name)
				} else {
					issueSummary[issue.Type] = &models.NamespaceErrorSummary{
						IssueType:    issue.Type,
						Count:        1,
						Description:  s.getIssueTypeDescription(issue.Type),
						AffectedPods: []string{pod.Name},
					}
				}

				switch issue.Severity {
				case "critical":
					report.CriticalIssuesCount++
				case "warning":
					report.WarningIssuesCount++
				}
			}
		}
	}

	report.HealthyPodsCount = report.TotalPodsAnalyzed - report.ProblematicPodsCount

	for _, summary := range issueSummary {
		report.Summary = append(report.Summary, *summary)
	}
	sort.Slice(report.Summary, func(i, j int) bool {
		return report.Summary[i].Count > report.Summary[j].Count
	})

	sort.Slice(report.ProblematicPods, func(i, j int) bool {
		iCritical := s.hasCriticalIssue(&report.ProblematicPods[i])
		jCritical := s.hasCriticalIssue(&report.ProblematicPods[j])
		if iCritical != jCritical {
			return iCritical
		}
		return report.ProblematicPods[i].RestartCount > report.ProblematicPods[j].RestartCount
	})

	s.logger.Info("namespace error analysis complete",
		"namespace", namespace,
		"totalPods", report.TotalPodsAnalyzed,
		"problematicPods", report.ProblematicPodsCount,
		"criticalIssues", report.CriticalIssuesCount,
		"warningIssues", report.WarningIssuesCount)

	return report, nil
}

func (s *namespaceService) filterPodsByOwner(pods []v1.Pod) []v1.Pod {
	filtered := []v1.Pod{}

	for i := range pods {
		pod := &pods[i]
		for _, owner := range pod.OwnerReferences {
			if owner.Kind == "ReplicaSet" || owner.Kind == "StatefulSet" {
				filtered = append(filtered, *pod)
				break
			}
		}
	}

	return filtered
}

func (s *namespaceService) analyzePod(ctx context.Context, pod *v1.Pod) *models.ProblematicPod {
	now := time.Now()
	age := now.Sub(pod.CreationTimestamp.Time)

	problematicPod := &models.ProblematicPod{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Phase:     string(pod.Status.Phase),
		Status:    pod.Status.Reason,
		Age:       age,
		AgeString: s.formatDuration(age),
		Issues:    []models.PodIssue{},
	}

	s.setOwnerInfo(pod, problematicPod)

	problematicPod.RestartCount = s.getTotalRestartCount(pod)

	if int(problematicPod.RestartCount) > s.podRestartThreshold {
		problematicPod.Issues = append(problematicPod.Issues, models.PodIssue{
			Type:        models.PodIssueHighRestarts,
			Description: fmt.Sprintf("Pod has restarted %d times (threshold: %d)", problematicPod.RestartCount, s.podRestartThreshold),
			Severity:    "critical",
			Details:     s.getRestartDetails(pod),
		})
	}

	if pod.Status.Phase == v1.PodPending && age > 5*time.Minute {
		issue := models.PodIssue{
			Type:        models.PodIssuePending,
			Description: fmt.Sprintf("Pod has been pending for %s", s.formatDuration(age)),
			Severity:    "critical",
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodScheduled && condition.Status == v1.ConditionFalse {
				issue.Details = condition.Message
				if strings.Contains(strings.ToLower(condition.Message), "insufficient") {
					issue.Type = models.PodIssueResourceConstraints
				} else if strings.Contains(strings.ToLower(condition.Message), "unschedulable") {
					issue.Type = models.PodIssueUnschedulable
				}
				break
			}
		}

		problematicPod.Issues = append(problematicPod.Issues, issue)
	}

	s.checkContainerStatuses(pod, problematicPod)

	if len(problematicPod.Issues) > 0 {
		events, err := s.getRecentPodEvents(ctx, pod)
		if err == nil {
			problematicPod.Events = events
		}
	}

	return problematicPod
}

func (s *namespaceService) setOwnerInfo(pod *v1.Pod, problematicPod *models.ProblematicPod) {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			problematicPod.OwnerKind = "Deployment"
			parts := strings.Split(owner.Name, "-")
			if len(parts) > 1 {
				problematicPod.OwnerName = strings.Join(parts[:len(parts)-1], "-")
			} else {
				problematicPod.OwnerName = owner.Name
			}
			break
		} else if owner.Kind == "StatefulSet" {
			problematicPod.OwnerKind = "StatefulSet"
			problematicPod.OwnerName = owner.Name
			break
		}
	}
}

func (s *namespaceService) getTotalRestartCount(pod *v1.Pod) int32 {
	var totalRestarts int32

	for i := range pod.Status.ContainerStatuses {
		status := &pod.Status.ContainerStatuses[i]
		totalRestarts += status.RestartCount
	}

	for i := range pod.Status.InitContainerStatuses {
		status := &pod.Status.InitContainerStatuses[i]
		totalRestarts += status.RestartCount
	}

	return totalRestarts
}

func (s *namespaceService) getRestartDetails(pod *v1.Pod) string {
	details := []string{}

	for i := range pod.Status.ContainerStatuses {
		status := &pod.Status.ContainerStatuses[i]
		if status.RestartCount > 0 {
			details = append(details, fmt.Sprintf("%s: %d restarts", status.Name, status.RestartCount))
		}
	}

	if len(details) > 0 {
		return strings.Join(details, ", ")
	}
	return ""
}

func (s *namespaceService) checkContainerStatuses(pod *v1.Pod, problematicPod *models.ProblematicPod) {
	for i := range pod.Status.ContainerStatuses {
		status := &pod.Status.ContainerStatuses[i]
		if status.State.Waiting != nil && status.State.Waiting.Reason == "CrashLoopBackOff" {
			problematicPod.Issues = append(problematicPod.Issues, models.PodIssue{
				Type:        models.PodIssueCrashLoop,
				Description: fmt.Sprintf("Container %s is in CrashLoopBackOff state", status.Name),
				Severity:    "critical",
				Details:     status.State.Waiting.Message,
			})
		}

		if status.State.Waiting != nil &&
			(status.State.Waiting.Reason == "ImagePullBackOff" || status.State.Waiting.Reason == "ErrImagePull") {
			problematicPod.Issues = append(problematicPod.Issues, models.PodIssue{
				Type:        models.PodIssueImagePull,
				Description: fmt.Sprintf("Container %s has image pull error: %s", status.Name, status.State.Waiting.Reason),
				Severity:    "critical",
				Details:     status.State.Waiting.Message,
			})
		}

		if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
			problematicPod.Issues = append(problematicPod.Issues, models.PodIssue{
				Type:        models.PodIssueFailed,
				Description: fmt.Sprintf("Container %s terminated with exit code %d", status.Name, status.State.Terminated.ExitCode),
				Severity:    "warning",
				Details:     status.State.Terminated.Reason,
			})
		}
	}
}

func (s *namespaceService) getRecentPodEvents(ctx context.Context, pod *v1.Pod) ([]models.EventInfo, error) {
	fieldSelector := fmt.Sprintf("involvedObject.kind=Pod,involvedObject.name=%s,involvedObject.namespace=%s",
		pod.Name, pod.Namespace)

	eventList, err := s.k8sClient.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, err
	}

	events := []models.EventInfo{}
	cutoff := time.Now().Add(-1 * time.Hour)

	for i := range eventList.Items {
		event := &eventList.Items[i]
		if event.LastTimestamp.After(cutoff) && event.Type == "Warning" {
			events = append(events, models.EventInfo{
				Type:           event.Type,
				Reason:         event.Reason,
				Message:        event.Message,
				FirstTimestamp: event.FirstTimestamp,
				LastTimestamp:  event.LastTimestamp,
				Count:          event.Count,
				Source:         fmt.Sprintf("%s/%s", event.Source.Component, event.Source.Host),
			})
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].LastTimestamp.After(events[j].LastTimestamp.Time)
	})

	if len(events) > 5 {
		events = events[:5]
	}

	return events, nil
}

func (s *namespaceService) hasCriticalIssue(pod *models.ProblematicPod) bool {
	for _, issue := range pod.Issues {
		if issue.Severity == "critical" {
			return true
		}
	}
	return false
}

func (s *namespaceService) getIssueTypeDescription(issueType models.PodIssueType) string {
	descriptions := map[models.PodIssueType]string{
		models.PodIssueHighRestarts:        "Pods with excessive restart counts",
		models.PodIssuePending:             "Pods stuck in pending state",
		models.PodIssueFailed:              "Pods in failed state",
		models.PodIssueCrashLoop:           "Pods in crash loop backoff",
		models.PodIssueImagePull:           "Pods with image pull errors",
		models.PodIssueResourceConstraints: "Pods with insufficient resources",
		models.PodIssueUnschedulable:       "Pods that cannot be scheduled",
	}

	if desc, ok := descriptions[issueType]; ok {
		return desc
	}
	return string(issueType)
}

func (s *namespaceService) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	return fmt.Sprintf("%dd", days)
}
