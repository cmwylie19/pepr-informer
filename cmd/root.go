package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/cmwylie19/pepr-informer/pkg/server"
	"github.com/spf13/cobra"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var logLevel string
var useInClusterConfig bool
var serverAddress string
var natsURL string

var (
	getInClusterConfig     = rest.InClusterConfig
	getConfigFromFlags     = clientcmd.BuildConfigFromFlags
	getDynamicNewForConfig = dynamic.NewForConfig
)

// rootCmd represents the base command when executed
var rootCmd = &cobra.Command{
	Use:   "pepr-informer",
	Short: "Starts the pepr-informer server",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Printf("[INFO] Starting pepr-informer...")

		// Determine Kubernetes config
		var config *rest.Config
		var err error

		if useInClusterConfig {
			config, err = getInClusterConfig()
			if err != nil {
				return fmt.Errorf("error building in-cluster config: %w", err)
			}
			log.Println("[INFO] Using in-cluster Kubernetes config")
		} else {
			config, err = getConfigFromFlags("", clientcmd.RecommendedHomeFile)
			if err != nil {
				return fmt.Errorf("error loading kubeconfig: %w", err)
			}
			log.Println("[INFO] Using kubeconfig from default location")
		}

		// Create Kubernetes dynamic client
		dynamicClient, err := getDynamicNewForConfig(config)
		if err != nil {
			return fmt.Errorf("error creating dynamic client: %w", err)
		}

		// Start HTTP server (Replaces gRPC)
		log.Printf("[INFO] Starting HTTP server on %s", serverAddress)
		server.StartHTTPServer(serverAddress, dynamicClient, config, natsURL)

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, error)")
	rootCmd.PersistentFlags().BoolVar(&useInClusterConfig, "in-cluster", true, "Use in-cluster Kubernetes configuration")
	rootCmd.PersistentFlags().StringVar(&serverAddress, "server-address", ":8080", "HTTP server listen address")
	rootCmd.PersistentFlags().StringVar(&natsURL, "nats-url", "nats://localhost:4222", "NATS server URL")
}

// Execute starts the CLI application
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Printf("[ERROR] CLI execution error: %v", err)
		os.Exit(1)
	}
}
