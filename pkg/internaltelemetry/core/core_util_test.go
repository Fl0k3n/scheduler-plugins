package core

import (
	"testing"

	"golang.org/x/exp/slices"
)

func TestScheduledNodes(t *testing.T) {
	tests := []struct{
		name     string
		first	 []ScheduledNode
		second   []ScheduledNode
		expected []ScheduledNode
	}{
		{
			name: "deployments of same node are merged",
			first: []ScheduledNode{{"n1", []string{"a"}}},
			second: []ScheduledNode{{"n1", []string{"b"}}},
			expected: []ScheduledNode{{"n1", []string{"a", "b"}}},
		},
		{
			name: "first can be empty",
			first: []ScheduledNode{{"n1", []string{"a"}}},
			second: []ScheduledNode{},
			expected: []ScheduledNode{{"n1", []string{"a"}}},
		},
		{
			name: "second can be empty",
			first: []ScheduledNode{},
			second: []ScheduledNode{{"n1", []string{"a"}}},
			expected: []ScheduledNode{{"n1", []string{"a"}}},
		},
		{
			name: "both can be empty",
			first: []ScheduledNode{},
			second: []ScheduledNode{},
			expected: []ScheduledNode{},
		},
		{
			name: "different nodes are merged",
			first: []ScheduledNode{
				{"n1", []string{"a", "c"}},
				{"n2", []string{"b"}},
				{"n3", []string{"a", "b"}},
			},
			second: []ScheduledNode{
				{"n1", []string{"a", "b"}},
				{"n2", []string{"a"}},
			},
			expected: []ScheduledNode{
				{"n1", []string{"a", "b", "c"}},
				{"n2", []string{"a", "b"}},
				{"n3", []string{"a", "b"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := mergeScheduledNodes(tt.first, tt.second)
			if len(merged) != len(tt.expected) {
				t.Fatalf("Invalid length of merged scheduled nodes, expected %d got %d", len(tt.expected), len(merged))
			}
			for _, esn := range tt.expected {
				idx := slices.IndexFunc(merged, func(sn ScheduledNode) bool {return sn.Name == esn.Name})
				if idx == -1 {
					t.Errorf("Node %s not found in merged", esn.Name)
				} else {
					msn := merged[idx]
					if len(esn.ScheduledDeployments) != len(msn.ScheduledDeployments) {
						t.Errorf("Invalid length of merged deployments for %s, expected %d got %d",
							esn.Name, len(esn.ScheduledDeployments), len(msn.ScheduledDeployments))
					} else {
						for _, sd := range esn.ScheduledDeployments {
							if slices.Index(msn.ScheduledDeployments, sd) == -1 {
								t.Errorf("Deployment %s should be scheduled on %s but isn't", sd, esn.Name)
							}
						}
					}
				}
			}
		})
	}
}
