package server

import (
	"testing"
)

func TestGetFormattedGV(t *testing.T) {
	tests := []struct {
		name     string
		group    string
		version  string
		expected string
	}{
		{
			name:     "Group is undefined",
			group:    "",
			version:  "v1",
			expected: "v1",
		},
		{
			name:     "Group and Version",
			group:    "apps",
			version:  "v1",
			expected: "apps/v1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := getFormattedGV(tc.group, tc.version)
			if actual != tc.expected {
				t.Errorf("expected: %s, got: %s", tc.expected, actual)
			}
		})
	}
}

// Test cases for generateNATSTopic function
func TestGenerateNATSTopic(t *testing.T) {
	tests := []struct {
		name     string
		req      WatchRequest
		expected string
	}{
		{
			name: "All fields present",
			req: WatchRequest{
				Group:     "apps",
				Version:   "v1",
				Resource:  "deployments",
				Namespace: "default",
			},
			expected: "k8s.apps.v1.deployments.default",
		},
		{
			name: "No Group",
			req: WatchRequest{
				Group:     "",
				Version:   "v1",
				Resource:  "pods",
				Namespace: "default",
			},
			expected: "k8s.v1.pods.default",
		},
		{
			name: "No Namespace",
			req: WatchRequest{
				Group:     "networking.k8s.io",
				Version:   "v1",
				Resource:  "ingresses",
				Namespace: "",
			},
			expected: "k8s.networking.k8s.io.v1.ingresses",
		},
		{
			name: "No Group and No Namespace",
			req: WatchRequest{
				Group:     "",
				Version:   "v1",
				Resource:  "services",
				Namespace: "",
			},
			expected: "k8s.v1.services",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := generateNATSTopic(tc.req)
			if result != tc.expected {
				t.Errorf("Expected topic: %s, got: %s", tc.expected, result)
			}
		})
	}
}
