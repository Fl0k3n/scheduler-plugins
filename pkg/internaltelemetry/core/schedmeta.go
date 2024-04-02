package core

import "slices"

type ScheduledNode struct {
	Name string
	ScheduledDeployments []string
}

func (s *ScheduledNode) DeepCopy() ScheduledNode {
	return ScheduledNode{
		Name: s.Name,
		ScheduledDeployments: slices.Clone(s.ScheduledDeployments),
	}
}

func (s *ScheduledNode) MarkScheduled(deplName string) {
	for _, sd := range s.ScheduledDeployments {
		if sd == deplName {
			return
		}
	}
	s.ScheduledDeployments = append(s.ScheduledDeployments, deplName)
}

func (s *ScheduledNode) HasScheduled(deplName string) bool {
	return slices.Contains(s.ScheduledDeployments, deplName)
}

// must exclude pod that is currently being scheduled
type QueuedPods struct {
	PerDeploymentCounts map[string]int // deploymentName -> number of remaining pods
}

func (q *QueuedPods) AllQueued() bool {
	for _, count := range q.PerDeploymentCounts {
		if count > 0 {
			return false
		}
	}
	return true
}