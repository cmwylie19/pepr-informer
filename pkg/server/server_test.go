package server

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"k8s.io/client-go/dynamic"

	"github.com/cmwylie19/pepr-informer/api"
	"github.com/cmwylie19/pepr-informer/mocks"
	"k8s.io/client-go/rest"
)

func TestNewServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLoggerInterface(ctrl)

	var mockClient dynamic.Interface
	mockRestConfig := &rest.Config{}
	s := NewServer(mockClient, mockRestConfig, mockLogger)

	if s == nil {
		t.Errorf("Expected server object, got nil")
	}

	if s.dynamicClient != mockClient {
		t.Errorf("Expected dynamicClient to be set")
	}

	if s.config != mockRestConfig {
		t.Errorf("Expected restConfig to be set")
	}

	if s.Logger != mockLogger {
		t.Errorf("Expected logger to be set")
	}
}

func TestFormSessionID(t *testing.T) {
	tests := []struct {
		name     string
		inputReq *api.WatchRequest
		expected string
	}{
		{
			name:     "Resource and group",
			inputReq: &api.WatchRequest{Resource: "pods", Group: "v1"},
			expected: "Group: v1, Version: , Resource: pods, Namespace: *",
		},
		{
			name:     "Resource only",
			inputReq: &api.WatchRequest{Resource: "pods"},
			expected: "Group: '', Version: , Resource: pods, Namespace: *",
		},
		{
			name:     "Group only",
			inputReq: &api.WatchRequest{Group: "v1"},
			expected: "Group: v1, Version: , Resource: , Namespace: *",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := formatSessionID(tc.inputReq)
			if actual != tc.expected {
				t.Errorf("expected: %s, got: %s", tc.expected, actual)
			}
		})
	}
}

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

func TestFormatRequest(t *testing.T) {
	mockGetResourceName := func(_ *rest.Config, group, version, resource string) (string, error) {
		mockResources := map[string]string{
			"pod":         "pods",
			"POD":         "pods",
			"deployments": "deployments",
			"services":    "services",
		}

		if plural, exists := mockResources[resource]; exists {
			return plural, nil
		}
		return "", fmt.Errorf("resource not found: %s", resource)
	}

	mockServer := &server{
		config:          nil,                 // Not needed for mock
		getResourceName: mockGetResourceName, // ✅ Inject mock function
	}

	tests := []struct {
		name     string
		inputReq *api.WatchRequest
		expected *api.WatchRequest
		wantErr  bool
	}{
		{
			name:     "Lowercase resource with missing plural",
			inputReq: &api.WatchRequest{Resource: "pod", Group: "v1"},
			expected: &api.WatchRequest{Resource: "pods", Group: "v1"},
			wantErr:  false,
		},
		{
			name:     "Uppercase resource with missing plural",
			inputReq: &api.WatchRequest{Resource: "POD", Group: "v1"},
			expected: &api.WatchRequest{Resource: "pods", Group: "v1"},
			wantErr:  false,
		},
		{
			name:     "Resource already plural",
			inputReq: &api.WatchRequest{Resource: "deployments"},
			expected: &api.WatchRequest{Resource: "deployments"},
			wantErr:  false,
		},
		{
			name:     "Resource with incorrect name",
			inputReq: &api.WatchRequest{Resource: "unknown"},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := mockServer.formatRequest(tc.inputReq)

			if (err != nil) != tc.wantErr {
				t.Errorf("expected error: %v, got error: %v", tc.wantErr, err)
			}

			if err == nil && !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("expected: %#v, got: %#v", tc.expected, actual)
			}
		})
	}
}
