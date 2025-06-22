package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/core/models"
)

type clusterIssuesService struct {
	clientset kubernetes.Interface
	logger    *slog.Logger
}

func NewClusterIssuesService(clientset kubernetes.Interface, logger *slog.Logger) core.ClusterIssuesService {
	return &clusterIssuesService{
		clientset: clientset,
		logger:    logger.With(slog.String("service", "cluster_issues")),
	}
}

func (s *clusterIssuesService) GetClusterIssues(ctx context.Context, namespace string, severityFilter string) (*models.ClusterIssues, error) {
	listOptions := metav1.ListOptions{}
	if namespace != "" && namespace != "all" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.namespace=%s", namespace)
	}

	pods, err := s.clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	issues := &models.ClusterIssues{
		TotalPods:         len(pods.Items),
		IssueCategories:   make(map[string]int),
		IssuesByNamespace: make(map[string]models.NamespaceIssues),
		CalculatedAt:      time.Now(),
	}

	allIssues := []models.ClusterPodIssue{}
	issuePatterns := make(map[string]*models.IssuePattern)

	cutoffTime1h := time.Now().Add(-1 * time.Hour)
	cutoffTime24h := time.Now().Add(-24 * time.Hour)

	for _, pod := range pods.Items {
		podIssues := s.analyzePod(&pod)

		if len(podIssues) == 0 {
			issues.HealthyPods++
			continue
		}

		issues.UnhealthyPods++

		nsIssues := issues.IssuesByNamespace[pod.Namespace]
		nsIssues.Namespace = pod.Namespace
		nsIssues.TotalPods++
		nsIssues.IssuesCount += len(podIssues)

		for _, issue := range podIssues {
			allIssues = append(allIssues, issue)

			issues.IssueCategories[issue.Category]++

			if issue.Severity == models.SeverityCritical {
				nsIssues.CriticalCount++
				issues.CriticalIssues = append(issues.CriticalIssues, issue)
			} else if issue.Severity == models.SeverityWarning {
				nsIssues.WarningCount++
			}

			if issue.LastSeen.After(cutoffTime1h) {
				issues.IssueVelocity.NewIssuesLastHour++
			}
			if issue.LastSeen.After(cutoffTime24h) {
				issues.IssueVelocity.NewIssuesLast24h++
			}

			s.detectPatterns(&pod, issue, issuePatterns)
		}

		if nsIssues.IssuesCount > 0 {
			nsIssues.TopIssues = append(nsIssues.TopIssues, podIssues...)
			issues.IssuesByNamespace[pod.Namespace] = nsIssues
		}
	}

	if severityFilter != "" {
		allIssues = s.filterBySeverity(allIssues, severityFilter)
	}

	s.calculateTopIssues(issues, allIssues)
	s.calculateIssueVelocity(issues)
	s.processPatterns(issues, issuePatterns)
	s.sortCriticalIssues(issues)

	return issues, nil
}

func (s *clusterIssuesService) analyzePod(pod *corev1.Pod) []models.ClusterPodIssue {
	issues := []models.ClusterPodIssue{}

	if pod.Status.Phase == corev1.PodPending {
		issue := s.analyzePendingPod(pod)
		if issue != nil {
			issues = append(issues, *issue)
		}
	}

	if pod.Status.Phase == corev1.PodFailed {
		issue := models.ClusterPodIssue{
			PodName:   pod.Name,
			Namespace: pod.Namespace,
			Category:  models.IssueCategoryFailed,
			Severity:  models.SeverityCritical,
			Reason:    string(pod.Status.Phase),
			Message:   pod.Status.Message,
			LastSeen:  time.Now(),
			NodeName:  pod.Spec.NodeName,
		}
		issues = append(issues, issue)
	}

	if pod.Status.Reason == "Evicted" {
		issue := models.ClusterPodIssue{
			PodName:   pod.Name,
			Namespace: pod.Namespace,
			Category:  models.IssueCategoryEvicted,
			Severity:  models.SeverityWarning,
			Reason:    pod.Status.Reason,
			Message:   pod.Status.Message,
			LastSeen:  time.Now(),
			NodeName:  pod.Spec.NodeName,
		}
		issues = append(issues, issue)
	}

	for _, status := range pod.Status.ContainerStatuses {
		containerIssues := s.analyzeContainerStatus(pod, &status)
		issues = append(issues, containerIssues...)
	}

	for _, status := range pod.Status.InitContainerStatuses {
		if status.State.Waiting != nil || (status.State.Terminated != nil && status.State.Terminated.ExitCode != 0) {
			issue := models.ClusterPodIssue{
				PodName:       pod.Name,
				Namespace:     pod.Namespace,
				Category:      models.IssueCategoryInitError,
				Severity:      models.SeverityCritical,
				ContainerName: status.Name,
				LastSeen:      time.Now(),
				NodeName:      pod.Spec.NodeName,
			}

			if status.State.Waiting != nil {
				issue.Reason = status.State.Waiting.Reason
				issue.Message = status.State.Waiting.Message
			} else if status.State.Terminated != nil {
				issue.Reason = status.State.Terminated.Reason
				issue.Message = fmt.Sprintf("Init container exited with code %d", status.State.Terminated.ExitCode)
			}

			issues = append(issues, issue)
		}
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
			if time.Since(condition.LastTransitionTime.Time) > 5*time.Minute {
				issue := models.ClusterPodIssue{
					PodName:   pod.Name,
					Namespace: pod.Namespace,
					Category:  models.IssueCategoryUnhealthy,
					Severity:  models.SeverityWarning,
					Reason:    "NotReady",
					Message:   fmt.Sprintf("Pod not ready for %s", time.Since(condition.LastTransitionTime.Time).Round(time.Minute)),
					LastSeen:  time.Now(),
					NodeName:  pod.Spec.NodeName,
				}
				issues = append(issues, issue)
			}
		}
	}

	return issues
}

func (s *clusterIssuesService) analyzePendingPod(pod *corev1.Pod) *models.ClusterPodIssue {
	if time.Since(pod.CreationTimestamp.Time) < 30*time.Second {
		return nil
	}

	issue := &models.ClusterPodIssue{
		PodName:   pod.Name,
		Namespace: pod.Namespace,
		Category:  models.IssueCategoryPending,
		Severity:  models.SeverityWarning,
		Reason:    "PendingScheduling",
		Message:   fmt.Sprintf("Pod pending for %s", time.Since(pod.CreationTimestamp.Time).Round(time.Minute)),
		LastSeen:  time.Now(),
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse {
			issue.Reason = condition.Reason
			issue.Message = condition.Message
			if strings.Contains(condition.Message, "Insufficient") {
				issue.Severity = models.SeverityCritical
			}
			break
		}
	}

	return issue
}

func (s *clusterIssuesService) analyzeContainerStatus(pod *corev1.Pod, status *corev1.ContainerStatus) []models.ClusterPodIssue {
	issues := []models.ClusterPodIssue{}

	if status.State.Waiting != nil {
		issue := models.ClusterPodIssue{
			PodName:       pod.Name,
			Namespace:     pod.Namespace,
			ContainerName: status.Name,
			Reason:        status.State.Waiting.Reason,
			Message:       status.State.Waiting.Message,
			LastSeen:      time.Now(),
			NodeName:      pod.Spec.NodeName,
		}

		switch status.State.Waiting.Reason {
		case "CrashLoopBackOff":
			issue.Category = models.IssueCategoryCrashLoop
			issue.Severity = models.SeverityCritical
			issue.IsRecurring = true
			issue.Count = int(status.RestartCount)
		case "ImagePullBackOff", "ErrImagePull":
			issue.Category = models.IssueCategoryImagePull
			issue.Severity = models.SeverityCritical
		case "CreateContainerConfigError":
			issue.Category = models.IssueCategoryConfigError
			issue.Severity = models.SeverityCritical
		default:
			issue.Category = models.IssueCategoryUnhealthy
			issue.Severity = models.SeverityWarning
		}

		issues = append(issues, issue)
	}

	if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
		issue := models.ClusterPodIssue{
			PodName:       pod.Name,
			Namespace:     pod.Namespace,
			ContainerName: status.Name,
			Reason:        status.State.Terminated.Reason,
			Message:       fmt.Sprintf("Container terminated with exit code %d", status.State.Terminated.ExitCode),
			LastSeen:      status.State.Terminated.FinishedAt.Time,
			NodeName:      pod.Spec.NodeName,
		}

		if status.State.Terminated.Reason == "OOMKilled" {
			issue.Category = models.IssueCategoryOOMKilled
			issue.Severity = models.SeverityCritical
			issue.Message = "Container killed due to Out Of Memory"
		} else {
			issue.Category = models.IssueCategoryFailed
			issue.Severity = models.SeverityWarning
		}

		issues = append(issues, issue)
	}

	if status.RestartCount > 5 && len(issues) == 0 {
		issue := models.ClusterPodIssue{
			PodName:       pod.Name,
			Namespace:     pod.Namespace,
			ContainerName: status.Name,
			Category:      models.IssueCategoryUnhealthy,
			Severity:      models.SeverityWarning,
			Reason:        "HighRestartCount",
			Message:       fmt.Sprintf("Container has restarted %d times", status.RestartCount),
			Count:         int(status.RestartCount),
			IsRecurring:   true,
			LastSeen:      time.Now(),
			NodeName:      pod.Spec.NodeName,
		}
		issues = append(issues, issue)
	}

	return issues
}

func (s *clusterIssuesService) detectPatterns(pod *corev1.Pod, issue models.ClusterPodIssue, patterns map[string]*models.IssuePattern) {
	patternKey := fmt.Sprintf("%s:%s", issue.Category, issue.Reason)

	if pattern, exists := patterns[patternKey]; exists {
		pattern.Count++
		pattern.LastSeen = time.Now()
		if !contains(pattern.Namespaces, pod.Namespace) {
			pattern.Namespaces = append(pattern.Namespaces, pod.Namespace)
		}

		for k, v := range pod.Labels {
			if pattern.CommonLabels[k] == v {
				continue
			} else {
				delete(pattern.CommonLabels, k)
			}
		}
	} else {
		commonLabels := make(map[string]string)
		for k, v := range pod.Labels {
			commonLabels[k] = v
		}

		patterns[patternKey] = &models.IssuePattern{
			Type:         issue.Category,
			Description:  fmt.Sprintf("%s: %s", issue.Category, issue.Reason),
			Count:        1,
			Namespaces:   []string{pod.Namespace},
			CommonLabels: commonLabels,
			FirstSeen:    time.Now(),
			LastSeen:     time.Now(),
		}
	}
}

func (s *clusterIssuesService) calculateTopIssues(issues *models.ClusterIssues, allIssues []models.ClusterPodIssue) {
	categoryCount := make(map[string]*models.IssueSummary)

	for _, issue := range allIssues {
		key := fmt.Sprintf("%s:%s", issue.Category, issue.Severity)
		if summary, exists := categoryCount[key]; exists {
			summary.Count++
			summary.AffectedPods = append(summary.AffectedPods, fmt.Sprintf("%s/%s", issue.Namespace, issue.PodName))
		} else {
			categoryCount[key] = &models.IssueSummary{
				Category:     issue.Category,
				Count:        1,
				Severity:     issue.Severity,
				Description:  s.getCategoryDescription(issue.Category),
				AffectedPods: []string{fmt.Sprintf("%s/%s", issue.Namespace, issue.PodName)},
			}
		}
	}

	for _, summary := range categoryCount {
		if len(summary.AffectedPods) > 5 {
			summary.AffectedPods = append(summary.AffectedPods[:5], fmt.Sprintf("... and %d more", len(summary.AffectedPods)-5))
		}
		issues.TopIssues = append(issues.TopIssues, *summary)
	}

	sort.Slice(issues.TopIssues, func(i, j int) bool {
		if issues.TopIssues[i].Severity == issues.TopIssues[j].Severity {
			return issues.TopIssues[i].Count > issues.TopIssues[j].Count
		}
		return s.getSeverityWeight(issues.TopIssues[i].Severity) > s.getSeverityWeight(issues.TopIssues[j].Severity)
	})

	if len(issues.TopIssues) > 10 {
		issues.TopIssues = issues.TopIssues[:10]
	}
}

func (s *clusterIssuesService) calculateIssueVelocity(issues *models.ClusterIssues) {
	velocity := &issues.IssueVelocity

	if velocity.NewIssuesLastHour > velocity.ResolvedLastHour {
		velocity.TrendDirection = models.TrendDegrading
	} else if velocity.NewIssuesLastHour < velocity.ResolvedLastHour {
		velocity.TrendDirection = models.TrendImproving
	} else {
		velocity.TrendDirection = models.TrendStable
	}

	if velocity.NewIssuesLast24h > 0 {
		velocity.VelocityPerHour = float64(velocity.NewIssuesLast24h) / 24.0
	}
}

func (s *clusterIssuesService) processPatterns(issues *models.ClusterIssues, patterns map[string]*models.IssuePattern) {
	for _, pattern := range patterns {
		if pattern.Count >= 3 {
			issues.Patterns = append(issues.Patterns, *pattern)
		}
	}

	sort.Slice(issues.Patterns, func(i, j int) bool {
		return issues.Patterns[i].Count > issues.Patterns[j].Count
	})

	if len(issues.Patterns) > 5 {
		issues.Patterns = issues.Patterns[:5]
	}
}

func (s *clusterIssuesService) sortCriticalIssues(issues *models.ClusterIssues) {
	sort.Slice(issues.CriticalIssues, func(i, j int) bool {
		return issues.CriticalIssues[i].LastSeen.After(issues.CriticalIssues[j].LastSeen)
	})

	if len(issues.CriticalIssues) > 20 {
		issues.CriticalIssues = issues.CriticalIssues[:20]
	}
}

func (s *clusterIssuesService) filterBySeverity(issues []models.ClusterPodIssue, severity string) []models.ClusterPodIssue {
	filtered := []models.ClusterPodIssue{}
	for _, issue := range issues {
		if issue.Severity == severity {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func (s *clusterIssuesService) getCategoryDescription(category string) string {
	descriptions := map[string]string{
		models.IssueCategoryCrashLoop:     "Container repeatedly crashing and restarting",
		models.IssueCategoryImagePull:     "Unable to pull container image",
		models.IssueCategoryPending:       "Pod unable to be scheduled",
		models.IssueCategoryOOMKilled:     "Container terminated due to memory limit",
		models.IssueCategoryEvicted:       "Pod evicted from node",
		models.IssueCategoryFailed:        "Pod in failed state",
		models.IssueCategoryUnhealthy:     "Pod health checks failing",
		models.IssueCategoryInitError:     "Init container failing to start",
		models.IssueCategoryVolumeMount:   "Volume mount issues",
		models.IssueCategoryConfigError:   "Configuration or secret mounting error",
		models.IssueCategoryNetworkError:  "Network connectivity issues",
		models.IssueCategoryResourceQuota: "Resource quota limits exceeded",
	}

	if desc, ok := descriptions[category]; ok {
		return desc
	}
	return "Unknown issue category"
}

func (s *clusterIssuesService) getSeverityWeight(severity string) int {
	switch severity {
	case models.SeverityCritical:
		return 3
	case models.SeverityWarning:
		return 2
	case models.SeverityInfo:
		return 1
	default:
		return 0
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

