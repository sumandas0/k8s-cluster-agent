package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sumandas0/k8s-cluster-agent/internal/core"
	"github.com/sumandas0/k8s-cluster-agent/internal/core/models"
)

type healthScoreService struct {
	clientset kubernetes.Interface
	logger    *slog.Logger
}

func NewHealthScoreService(clientset kubernetes.Interface, logger *slog.Logger) core.HealthScoreService {
	return &healthScoreService{
		clientset: clientset,
		logger:    logger.With(slog.String("service", "health_score")),
	}
}

func (s *healthScoreService) CalculateHealthScore(ctx context.Context, namespace, podName string) (*models.PodHealthScore, error) {
	pod, err := s.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	events, err := s.getPodEvents(ctx, namespace, podName)
	if err != nil {
		s.logger.Error("failed to get pod events", slog.String("error", err.Error()))
		events = &corev1.EventList{Items: []corev1.Event{}}
	}

	healthScore := &models.PodHealthScore{
		PodName:      podName,
		Namespace:    namespace,
		CalculatedAt: time.Now(),
		Components:   make(map[string]models.HealthComponent),
		Details:      s.extractHealthDetails(pod, events),
	}

	s.calculateRestartScore(healthScore, pod)
	s.calculateContainerStateScore(healthScore, pod)
	s.calculateEventScore(healthScore, events)
	s.calculatePodConditionScore(healthScore, pod)
	s.calculateUptimeScore(healthScore, pod)

	healthScore.OverallScore = s.calculateOverallScore(healthScore.Components)
	healthScore.Status = healthScore.GetHealthStatus()

	return healthScore, nil
}

func (s *healthScoreService) calculateRestartScore(score *models.PodHealthScore, pod *corev1.Pod) {
	totalRestarts := int32(0)
	for _, status := range pod.Status.ContainerStatuses {
		totalRestarts += status.RestartCount
	}

	var restartScore int
	switch {
	case totalRestarts == 0:
		restartScore = 100
	case totalRestarts <= 2:
		restartScore = 85
	case totalRestarts <= 5:
		restartScore = 70
	case totalRestarts <= 10:
		restartScore = 50
	case totalRestarts <= 20:
		restartScore = 30
	default:
		restartScore = 10
	}

	podAge := time.Since(pod.CreationTimestamp.Time)
	if podAge > 0 && totalRestarts > 0 {
		restartsPerHour := float64(totalRestarts) / podAge.Hours()
		if restartsPerHour > 1 {
			restartScore = int(math.Max(float64(restartScore)*0.5, 10))
		}
		score.Details.RestartFrequency = fmt.Sprintf("%.2f restarts/hour", restartsPerHour)
	}

	score.Components["restarts"] = models.HealthComponent{
		Name:        "Container Restarts",
		Score:       restartScore,
		Weight:      0.30,
		Status:      getComponentStatus(restartScore),
		Description: fmt.Sprintf("%d total restarts", totalRestarts),
	}
}

func (s *healthScoreService) calculateContainerStateScore(score *models.PodHealthScore, pod *corev1.Pod) {
	stateScore := 100
	unhealthyContainers := 0

	for _, status := range pod.Status.ContainerStatuses {
		containerHealth := models.ContainerHealth{
			Name:         status.Name,
			Ready:        status.Ready,
			RestartCount: status.RestartCount,
		}

		if status.State.Running != nil {
			containerHealth.State = "Running"
		} else if status.State.Waiting != nil {
			containerHealth.State = "Waiting"
			containerHealth.Reason = status.State.Waiting.Reason
			unhealthyContainers++
			switch status.State.Waiting.Reason {
			case "CrashLoopBackOff", "Error":
				stateScore = int(math.Min(float64(stateScore), 20))
			case "ImagePullBackOff", "ErrImagePull":
				stateScore = int(math.Min(float64(stateScore), 30))
			default:
				stateScore = int(math.Min(float64(stateScore), 50))
			}
		} else if status.State.Terminated != nil {
			containerHealth.State = "Terminated"
			containerHealth.Reason = status.State.Terminated.Reason
			containerHealth.ExitCode = &status.State.Terminated.ExitCode
			unhealthyContainers++
			if status.State.Terminated.ExitCode != 0 {
				stateScore = int(math.Min(float64(stateScore), 40))
			}
		}

		score.Details.ContainerStatuses = append(score.Details.ContainerStatuses, containerHealth)
	}

	if unhealthyContainers == 0 && len(pod.Status.ContainerStatuses) > 0 {
		readyCount := 0
		for _, status := range pod.Status.ContainerStatuses {
			if status.Ready {
				readyCount++
			}
		}
		readyPercentage := float64(readyCount) / float64(len(pod.Status.ContainerStatuses))
		stateScore = int(readyPercentage * 100)
	}

	score.Components["containerStates"] = models.HealthComponent{
		Name:        "Container States",
		Score:       stateScore,
		Weight:      0.25,
		Status:      getComponentStatus(stateScore),
		Description: fmt.Sprintf("%d/%d containers healthy", len(pod.Status.ContainerStatuses)-unhealthyContainers, len(pod.Status.ContainerStatuses)),
	}
}

func (s *healthScoreService) calculateEventScore(score *models.PodHealthScore, events *corev1.EventList) {
	eventScore := 100
	warningCount := 0
	recentEvents := make(map[string]*models.EventSummary)

	cutoffTime := time.Now().Add(-24 * time.Hour)

	for _, event := range events.Items {
		if event.LastTimestamp.Time.Before(cutoffTime) {
			continue
		}

		key := fmt.Sprintf("%s:%s", event.Type, event.Reason)
		if summary, exists := recentEvents[key]; exists {
			summary.Count += event.Count
			if event.LastTimestamp.Time.After(summary.LastSeen) {
				summary.LastSeen = event.LastTimestamp.Time
			}
		} else {
			recentEvents[key] = &models.EventSummary{
				Type:     event.Type,
				Reason:   event.Reason,
				Message:  event.Message,
				Count:    event.Count,
				LastSeen: event.LastTimestamp.Time,
			}
		}

		if event.Type == corev1.EventTypeWarning {
			warningCount++
			switch event.Reason {
			case "Failed", "FailedScheduling", "FailedMount":
				eventScore = int(math.Min(float64(eventScore), 30))
			case "BackOff", "CrashLoopBackOff":
				eventScore = int(math.Min(float64(eventScore), 40))
			case "Unhealthy":
				eventScore = int(math.Min(float64(eventScore), 50))
			default:
				eventScore = int(math.Min(float64(eventScore), 70))
			}
		}
	}

	for _, summary := range recentEvents {
		score.Details.RecentEvents = append(score.Details.RecentEvents, *summary)
	}

	score.Components["events"] = models.HealthComponent{
		Name:        "Recent Events",
		Score:       eventScore,
		Weight:      0.20,
		Status:      getComponentStatus(eventScore),
		Description: fmt.Sprintf("%d warning events in last 24h", warningCount),
	}
}

func (s *healthScoreService) calculatePodConditionScore(score *models.PodHealthScore, pod *corev1.Pod) {
	conditionScore := 100
	failedConditions := 0

	for _, condition := range pod.Status.Conditions {
		condStatus := models.ConditionStatus{
			Type:    string(condition.Type),
			Status:  string(condition.Status),
			Reason:  condition.Reason,
			Message: condition.Message,
		}
		score.Details.PodConditions = append(score.Details.PodConditions, condStatus)

		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case corev1.PodReady:
				conditionScore = int(math.Min(float64(conditionScore), 50))
				failedConditions++
			case corev1.PodScheduled:
				conditionScore = int(math.Min(float64(conditionScore), 30))
				failedConditions++
			case corev1.ContainersReady:
				conditionScore = int(math.Min(float64(conditionScore), 60))
				failedConditions++
			case corev1.PodInitialized:
				conditionScore = int(math.Min(float64(conditionScore), 70))
				failedConditions++
			}
		}
	}

	score.Components["conditions"] = models.HealthComponent{
		Name:        "Pod Conditions",
		Score:       conditionScore,
		Weight:      0.15,
		Status:      getComponentStatus(conditionScore),
		Description: fmt.Sprintf("%d/%d conditions healthy", len(pod.Status.Conditions)-failedConditions, len(pod.Status.Conditions)),
	}
}

func (s *healthScoreService) calculateUptimeScore(score *models.PodHealthScore, pod *corev1.Pod) {
	uptimeScore := 100
	podAge := time.Since(pod.CreationTimestamp.Time)
	score.Details.Uptime = formatDuration(podAge)

	if len(pod.Status.ContainerStatuses) > 0 {
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Running != nil {
				containerUptime := time.Since(status.State.Running.StartedAt.Time)
				uptimeRatio := containerUptime.Seconds() / podAge.Seconds()

				if uptimeRatio < 0.5 {
					uptimeScore = int(math.Min(float64(uptimeScore), 50))
				} else if uptimeRatio < 0.8 {
					uptimeScore = int(math.Min(float64(uptimeScore), 70))
				} else if uptimeRatio < 0.95 {
					uptimeScore = int(math.Min(float64(uptimeScore), 85))
				}
			}

			if status.LastTerminationState.Terminated != nil {
				lastRestart := status.LastTerminationState.Terminated.FinishedAt.Time
				score.Details.LastRestartTime = &lastRestart
				score.Details.LastRestartReason = status.LastTerminationState.Terminated.Reason
			}
		}
	}

	score.Components["uptime"] = models.HealthComponent{
		Name:        "Uptime/Stability",
		Score:       uptimeScore,
		Weight:      0.10,
		Status:      getComponentStatus(uptimeScore),
		Description: fmt.Sprintf("Pod age: %s", score.Details.Uptime),
	}
}

func (s *healthScoreService) calculateOverallScore(components map[string]models.HealthComponent) int {
	weightedSum := 0.0
	totalWeight := 0.0

	for _, component := range components {
		weightedSum += float64(component.Score) * component.Weight
		totalWeight += component.Weight
	}

	if totalWeight == 0 {
		return 0
	}

	return int(math.Round(weightedSum / totalWeight))
}

func (s *healthScoreService) getPodEvents(ctx context.Context, namespace, podName string) (*corev1.EventList, error) {
	fieldSelector := fmt.Sprintf("involvedObject.kind=Pod,involvedObject.name=%s", podName)
	return s.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
}

func (s *healthScoreService) extractHealthDetails(pod *corev1.Pod, _ *corev1.EventList) models.HealthDetails {
	details := models.HealthDetails{
		RestartCount:      0,
		ContainerStatuses: []models.ContainerHealth{},
		RecentEvents:      []models.EventSummary{},
		PodConditions:     []models.ConditionStatus{},
	}

	for _, status := range pod.Status.ContainerStatuses {
		details.RestartCount += status.RestartCount
	}

	return details
}

func getComponentStatus(score int) string {
	switch {
	case score >= 90:
		return "Excellent"
	case score >= 70:
		return "Good"
	case score >= 50:
		return "Fair"
	case score >= 30:
		return "Poor"
	default:
		return "Critical"
	}
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
