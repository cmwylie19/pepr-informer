package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// WatchRequest defines the input for a watch request
type WatchRequest struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Resource  string `json:"resource"`
	Namespace string `json:"namespace"`
}

// WatchResponse represents a Kubernetes event
type WatchResponse struct {
	EventType string `json:"eventType"`
	Details   string `json:"details"`
}

// Server manages Kubernetes watch events and sends them to NATS
type Server struct {
	dynamicClient dynamic.Interface
	config        *rest.Config
	natsConn      *nats.Conn
}

// NewServer initializes a new server instance with NATS
func NewServer(dynamicClient dynamic.Interface, restConfig *rest.Config, natsURL string) (*Server, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %v", err)
	}

	return &Server{
		dynamicClient: dynamicClient,
		config:        restConfig,
		natsConn:      nc,
	}, nil
}

// WatchHandler handles incoming HTTP watch requests
func (s *Server) WatchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	go func() {
		if err := s.Watch(req); err != nil {
			log.Printf("[ERROR] Watch failed: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"natsTopic": generateNATSTopic(req),
	})
}

// Watch starts a Kubernetes informer and publishes events to NATS
func (s *Server) Watch(req WatchRequest) error {
	req, err := s.formatRequest(req)
	if err != nil {
		return fmt.Errorf("failed to format request: %v", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    req.Group,
		Version:  req.Version,
		Resource: req.Resource,
	}

	log.Printf("[INFO] Starting watch for %s/%s/%s in namespace %s", req.Group, req.Version, req.Resource, req.Namespace)
	log.Printf("[DEBUG] GVR: %v", gvr)

	if s.dynamicClient == nil {
		return fmt.Errorf("dynamic client is not initialized")
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(s.dynamicClient, 5*time.Minute, req.Namespace, nil)
	informer := factory.ForResource(gvr).Informer()

	// Define event handlers with a single topic per resource
	eventProcessor := s.processEvent(req)
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { eventProcessor(obj, "ADD") },
		UpdateFunc: func(_, newObj interface{}) { eventProcessor(newObj, "UPDATE") },
		DeleteFunc: func(obj interface{}) { eventProcessor(obj, "DELETE") },
	})

	go informer.Run(make(chan struct{}))
	return nil
}

// formatRequest ensures the resource name is properly formatted
func (s *Server) formatRequest(req WatchRequest) (WatchRequest, error) {
	req.Resource = strings.ToLower(req.Resource)

	resourceName, err := s.getResourceName(s.config, req.Group, req.Version, req.Resource)
	if err != nil {
		return req, err
	}

	req.Resource = resourceName
	return req, nil
}

// getResourceName fetches the correct Kubernetes resource name
func (s *Server) getResourceName(restConfig *rest.Config, group, version, resource string) (string, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create discovery client: %w", err)
	}

	formattedGV := getFormattedGV(group, version)
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion(formattedGV)
	if err != nil {
		return "", fmt.Errorf("failed to fetch resource list: %w", err)
	}

	for _, apiResource := range resourceList.APIResources {
		if apiResource.SingularName == resource || apiResource.Name == resource {
			return apiResource.Name, nil
		}
	}

	return "", fmt.Errorf("resource not found: %s", resource)
}

// getFormattedGV formats the group and version
func getFormattedGV(group, version string) string {
	if group == "" {
		return version
	}
	return fmt.Sprintf("%s/%s", group, version)
}

// processEvent handles Kubernetes events and sends them to the appropriate NATS topic
func (s *Server) processEvent(req WatchRequest) func(interface{}, string) {
	return func(obj interface{}, eventType string) {
		jsonObj, err := toJson(obj)
		if err != nil {
			log.Printf("[ERROR] JSON conversion failed: %v", err)
			return
		}

		log.Printf("[INFO] EventType: %s, Details: %s", eventType, jsonObj)
		s.publishToNATS(req, eventType, jsonObj)
	}
}

// publishToNATS sends all events for a specific WatchRequest to a single topic
func (s *Server) publishToNATS(req WatchRequest, eventType, jsonData string) {
	topic := generateNATSTopic(req)

	event := WatchResponse{
		EventType: eventType,
		Details:   jsonData,
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal event: %v", err)
		return
	}

	if err := s.natsConn.Publish(topic, data); err != nil {
		log.Printf("[ERROR] Failed to publish event to topic %s: %v", topic, err)
	} else {
		log.Printf("[INFO] Event published to NATS topic: %s", topic)
	}
}

// toJson converts an object to a JSON string
func toJson(obj interface{}) (string, error) {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("error converting to JSON: %v", err)
	}
	return string(jsonData), nil
}

// StartHTTPServer starts an HTTP server to handle incoming watch requests
func StartHTTPServer(address string, dynamicClient dynamic.Interface, restConfig *rest.Config, natsURL string) {
	s, err := NewServer(dynamicClient, restConfig, natsURL)
	if err != nil {
		log.Fatalf("[ERROR] Failed to create server: %v", err)
	}

	http.HandleFunc("/watch", s.WatchHandler)

	log.Printf("[INFO] HTTP server listening on %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

// generateNATSTopic creates a NATS topic string dynamically
func generateNATSTopic(req WatchRequest) string {
	parts := []string{"k8s"}

	if req.Group != "" {
		parts = append(parts, strings.ToLower(req.Group))
	}
	parts = append(parts, strings.ToLower(req.Version), strings.ToLower(req.Resource))

	if req.Namespace != "" {
		parts = append(parts, strings.ToLower(req.Namespace))
	}

	return strings.Join(parts, ".")
}
