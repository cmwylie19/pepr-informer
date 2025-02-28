//go:build e2e
// +build e2e

package e2e_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var kubeConfigPath string

var _ = BeforeSuite(func() {
	kubeConfigPath = setupKindCluster()
	buildInformerImage()
	importInformerImage()

	buildCurlPod()
	importCurlPod()
	deployApplication(kubeConfigPath)
	deployCurlPod(kubeConfigPath)
	time.Sleep(10 * time.Second)
	waitForPods(kubeConfigPath)
})

var _ = AfterSuite(func() {
	teardownKindCluster(kubeConfigPath)
})
var namespace = "pepr-informer"
var _ = Describe("E2E Test", func() {
	Context("When deploying the application", func() {
		It("should create gRPC stream of watch events", func() {
			timeout := 5 * time.Second

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			cmd := exec.CommandContext(ctx, "kubectl", "exec", "curler", "-n", namespace, "--kubeconfig", kubeConfigPath, "--", "grpcurl", "-plaintext", "-d", `{"group": "", "version": "v1", "resource": "pod", "namespace": "pepr-informer"}`, "pepr-informer.pepr-informer.svc.cluster.local:50051", "api.WatchService.Watch")
			var cmdOut bytes.Buffer
			cmd.Stdout = &cmdOut
			cmd.Stderr = &cmdOut

			err := cmd.Run()

			if ctx.Err() == context.DeadlineExceeded {
				// Command timed out, but still process the partial output
				GinkgoWriter.Write([]byte("Command timed out but processing output...\n"))
			} else if err != nil {
				// Command finished with an error
				GinkgoWriter.Write(cmdOut.Bytes())
				Expect(err).NotTo(HaveOccurred(), "Failed to create gRPC stream of watch events")
			}

			// Process the output regardless of whether it was timed out or successful
			Expect(cmdOut.String()).To(ContainSubstring("ADD"))
			Expect(cmdOut.String()).To(ContainSubstring("Pod"))
		})

		It("should produce watch logs from the server", func() {
			podLogs := getPodLogs(kubeConfigPath)
			Expect(podLogs).To(ContainSubstring("Server listening at :50051"))
			Expect(podLogs).To(ContainSubstring("Starting watch for Group: '', Version: v1, Resource: pods, Namespace: pepr-informer"))
			Expect(podLogs).To(ContainSubstring("ADD"))
			Expect(podLogs).To(ContainSubstring("Pod"))
		})
	})
})

func deployCurlPod(kubeConfigPath string) {
	cmd := exec.Command("kubectl", "apply", "-f", "../hack/pod.yaml", "--kubeconfig", kubeConfigPath)
	var cmdOut bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdOut
	err := cmd.Run()
	if err != nil {
		GinkgoWriter.Write(cmdOut.Bytes())
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the curler pod")
	}
}
func waitForPods(kubeConfigPath string) {
	cmd0 := exec.Command("kubectl", "wait", "--for=condition=ready", "pod", "-n", namespace, "-l", "app=pepr-informer", "--kubeconfig", kubeConfigPath)
	var cmd0Out bytes.Buffer
	cmd0.Stdout = &cmd0Out
	cmd0.Stderr = &cmd0Out
	err := cmd0.Run()
	if err != nil {
		GinkgoWriter.Write(cmd0Out.Bytes())
		Expect(err).NotTo(HaveOccurred(), "Failed to wait for pepr-informer pod to be ready")
	}

	cmd1 := exec.Command("kubectl", "wait", "--for=condition=ready", "pod", "-n", namespace, "-l", "app=curler", "--kubeconfig", kubeConfigPath)
	var cmd1Out bytes.Buffer
	cmd1.Stdout = &cmd1Out
	cmd1.Stderr = &cmd1Out
	err = cmd1.Run()
	if err != nil {
		GinkgoWriter.Write(cmd1Out.Bytes())
		Expect(err).NotTo(HaveOccurred(), "Failed to wait for curler pod to be ready")
	}

}
func importCurlPod() {
	cmd := exec.Command("kind", "load", "docker-image", "curler:ci")
	var cmdOut bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdOut
	err := cmd.Run()
	if err != nil {
		GinkgoWriter.Write(cmdOut.Bytes())
	}
}
func buildCurlPod() {
	cmd := exec.Command("docker", "build", "-t", "curler:ci", "-f", "../hack/Dockerfile", "../hack")
	var cmdOut bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdOut
	err := cmd.Run()
	if err != nil {
		GinkgoWriter.Write(cmdOut.Bytes())
	}
}

func buildInformerImage() {
	cmd := exec.Command("docker", "build", "-t", "pepr-informer:ci", "../", "-f", "../Dockerfile")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		GinkgoWriter.Write(out.Bytes())
		Expect(err).NotTo(HaveOccurred(), "Failed to build pepr-informer image")
	}
}

func importInformerImage() {
	cmd := exec.Command("kind", "load", "docker-image", "pepr-informer:ci")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		GinkgoWriter.Write(out.Bytes())
		Expect(err).NotTo(HaveOccurred(), "Failed to import the pepr-informer image into kind")
	}
}

func setupKindCluster() string {
	cmd := exec.Command("kind", "create", "cluster")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		GinkgoWriter.Write(out.Bytes())
		Expect(err).NotTo(HaveOccurred(), "Failed to create the kind cluster")
	}

	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"
	return kubeConfigPath
}

func teardownKindCluster(kubeConfigPath string) {
	cmd := exec.Command("kind", "delete", "cluster", "--kubeconfig", kubeConfigPath)
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())

}

func deployApplication(kubeConfigPath string) {
	cmd := exec.Command("kubectl", "apply", "-k", "../kustomize/overlays/ci", "--kubeconfig", kubeConfigPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		GinkgoWriter.Write(out.Bytes())
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the pepr-informer")
	}
}

func getPodLogs(kubeConfigPath string) string {
	cmd := exec.Command("kubectl", "logs", "deploy/pepr-informer", "-n", namespace, "--kubeconfig", kubeConfigPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
	return out.String()
}
