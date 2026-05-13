package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/viper"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Config struct {
	Measurements []Measurement `mapstructure:"measurements"`
}

type Measurement struct {
	Name          string        `mapstructure:"name"`
	Client        ClientServer  `mapstructure:"client"`
	LoadGenerator LoadGenerator `mapstructure:"loadGenerator"`
	Turncat       Turncat       `mapstructure:"turncat"`
}

type ClientServer struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type LoadGenerator struct {
	Command string   `mapstructure:"command"`
	Args    []string `mapstructure:"args"`
}

type Turncat struct {
	Log               string `mapstructure:"log"`
	ClientAddress     string `mapstructure:"clientAddress"`
	TurnServerAddress string `mapstructure:"turnServerAddress"`
	PeerHostAddress   string `mapstructure:"peerHostAddress"`
}

func main() {

	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	kubeCfg, kubeErr := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if kubeErr != nil {
		panic(kubeErr.Error())
	}

	// create the clientset
	clientset, kubeErr := kubernetes.NewForConfig(kubeCfg)
	if kubeErr != nil {
		panic(kubeErr.Error())
	}

	// Dynamic client (for Gateway CRD)
	dynClient, err := dynamic.NewForConfig(kubeCfg)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// Set up viper to read the config file
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	var cfg Config

	// Find and read the config file
	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	// Unmarshal the config into the struct
	err = viper.Unmarshal(&cfg)
	if err != nil {
		panic(fmt.Errorf("fatal error unmarshaling file: %w", err))
	}

	var wg sync.WaitGroup

	// Loop through the measurements and get the necessary information for each measurement
	for _, measurement := range cfg.Measurements {

		peerHostAddress := "udp://"
		turnServerAddress := "turn://user-1:pass-1@"

		//----------------------------------------
		// TURN SERVER IP
		//----------------------------------------

		svc, err := clientset.CoreV1().
			Services("stunner").
			Get(ctx, "udp-gateway", metav1.GetOptions{})
		if err != nil {
			panic(err)
		}

		turnIP := svc.Status.LoadBalancer.Ingress[0].IP

		turnServerAddress += fmt.Sprintf("%s:", turnIP)

		//----------------------------------------
		// TURN SERVER PORT FROM GATEWAY
		//----------------------------------------

		gvr := schema.GroupVersionResource{
			Group:    "gateway.networking.k8s.io",
			Version:  "v1",
			Resource: "gateways",
		}

		gateway, err := dynClient.Resource(gvr).Namespace("stunner").Get(
			context.TODO(),
			"udp-gateway",
			metav1.GetOptions{},
		)
		if err != nil {
			panic(err)
		}

		turnPort := ""
		// Navigate the unstructured object
		listeners, found, err := unstructured.NestedSlice(gateway.Object, "spec", "listeners")
		if err != nil || !found {
			panic("listeners not found")
		}

		for _, listener := range listeners {
			listener, ok := listener.(map[string]interface{})
			if !ok {
				continue
			}
			if listener["name"] == "udp-listener" {
				// Convert to string
				turnPort = fmt.Sprintf("%d", listener["port"].(int64))
				break
			}
		}

		turnServerAddress += fmt.Sprintf("%s?transport=udp", turnPort)

		//----------------------------------------
		// IPERF SERVER CLUSTER IP
		//----------------------------------------
		iperfSvc, err := clientset.CoreV1().
			Services("default").
			Get(ctx, "iperf-server", metav1.GetOptions{})
		if err != nil {
			panic(err)
		}

		peerIP := iperfSvc.Spec.ClusterIP

		peerHostAddress += fmt.Sprintf("%s:5001", peerIP)

		measurement.Turncat.PeerHostAddress = peerHostAddress
		measurement.Turncat.TurnServerAddress = turnServerAddress

		// Start in a separate goroutine to allow multiple measurements to run concurrently
		wg.Add(1)
		go startMeasurement(measurement, &wg)
	}

	wg.Wait()
}

func startMeasurement(measurement Measurement, wg *sync.WaitGroup) {
	defer wg.Done()
	// This function will start the measurement based on the configuration
	// It will start turncat and the load generator with the appropriate parameters

	clientAddress := fmt.Sprintf("udp://%s:%d", measurement.Client.Host, measurement.Client.Port)

	fmt.Printf("Measurement Name: %s\n", measurement.Name)
	fmt.Printf("Hosted Client address: %s:%d\n", measurement.Client.Host, measurement.Client.Port)
	fmt.Printf("Peer Address: %s\n", measurement.Turncat.PeerHostAddress)
	fmt.Printf("Turn server address: %s\n", measurement.Turncat.TurnServerAddress)
	fmt.Print("------------------------------\n")

	fmt.Print("Starting turncat\n")

	// Get the authentication information for turncat

	turncat := exec.Command("turncat", "--log=all:INFO", clientAddress, measurement.Turncat.TurnServerAddress, measurement.Turncat.PeerHostAddress)
	turncat.Stdout = os.Stdout

	// Save start time
	startTime := time.Now()
	fmt.Printf("Measurement start time: %s\n", startTime.Format(time.RFC3339))

	// Print loadGenerator
	fmt.Printf("Load generator command: %s\n", measurement.LoadGenerator)

	fmt.Printf("Starting load generator with command: %s and args: %v\n", measurement.LoadGenerator.Command, measurement.LoadGenerator.Args)

	loadGenerator := exec.Command(measurement.LoadGenerator.Command, measurement.LoadGenerator.Args...)
	loadGenerator.Stdout = os.Stdout

	err := turncat.Start()
	if err != nil {
		panic(fmt.Errorf("fatal error starting turncat: %w", err))
	}

	err = loadGenerator.Start()
	if err != nil {
		panic(fmt.Errorf("fatal error starting load generator '%s': %w", measurement.LoadGenerator.Command, err))
	}

	loadGenerator.Wait()

	// Save end time
	endTime := time.Now()
	fmt.Printf("Measurement end time: %s\n", endTime.Format(time.RFC3339))

	fmt.Printf("Shutting down turncat for measurement: %s\n", measurement.Name)
	turncat.Process.Kill()

	// Run styx to save prometheus data
	fmt.Printf("Fetching prometheus data for measurement: %s\n", measurement.Name)
	bufferTime := 5 * time.Minute
	savePrometheusData(measurement.Name, startTime, endTime, bufferTime)

	fmt.Printf("Measurement completed: %s\n", measurement.Name)
}

func savePrometheusData(measurementName string, startTime, endTime time.Time, bufferTime time.Duration) {
	// Create output directory if it doesn't exist
	outputDir := filepath.Join("results", measurementName)
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return
	}

	// Get time offset between local system and Prometheus server
	timeOffset, err := getPrometheusTimeOffset()
	if err != nil {
		fmt.Printf("Warning: Could not determine time offset: %v. Using local time.\n", err)
		timeOffset = 0
	}

	// Apply time offset to start and end times
	adjustedStart := startTime.Add(timeOffset)
	adjustedEnd := endTime.Add(timeOffset)

	// Add buffer time before start and after end to capture metrics
	bufferedStart := adjustedStart.Add(-bufferTime)

	// Format start time in UTC for styx
	startStr := bufferedStart.UTC().Format("2006-01-02T15:04:05")

	// Calculate duration
	duration := adjustedEnd.Sub(bufferedStart)
	durationStr := duration.String()

	fmt.Printf("Running styx for start: %s, duration: %s (adjusted by offset: %v)\n", startStr, durationStr, timeOffset)

	// CPU query
	cpuQuery := "sum(rate(container_cpu_usage_seconds_total[5m])) by (namespace)"
	cpuFile := filepath.Join(outputDir, "cpu_by_namespace.csv")
	err = runStyxQuery(cpuQuery, startStr, durationStr, cpuFile)
	if err != nil {
		fmt.Printf("Error fetching CPU data: %v\n", err)
	}

	// Memory query
	memQuery := "sum(container_memory_working_set_bytes) by (namespace) / 1024 / 1024 / 1024"
	memFile := filepath.Join(outputDir, "memory_by_namespace.csv")
	err = runStyxQuery(memQuery, startStr, durationStr, memFile)
	if err != nil {
		fmt.Printf("Error fetching memory data: %v\n", err)
	}

	fmt.Printf("Prometheus data saved to %s\n", outputDir)
}

func runStyxQuery(query, start, duration, outputFile string) error {
	// Run styx with the specified query and time range
	cmd := exec.Command("styx", "--start="+start, "--duration="+duration, query)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Styx output:\n%s\n", string(output))
		return fmt.Errorf("error running styx: %w", err)
	}

	// Write output to file
	err = os.WriteFile(outputFile, output, 0644)
	if err != nil {
		return fmt.Errorf("error writing output file: %w", err)
	}

	fmt.Printf("Saved query results to %s\n", outputFile)
	return nil
}

func getPrometheusTimeOffset() (time.Duration, error) {
	// Try to get time from Prometheus HTTP API
	resp, err := http.Get("http://localhost:9090/api/v1/query?query=time()")
	if err != nil {
		return 0, fmt.Errorf("error connecting to prometheus: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading prometheus response: %w", err)
	}

	// Parse JSON response
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return 0, fmt.Errorf("error parsing prometheus response: %w", err)
	}

	// Extract timestamp from response
	if data, ok := result["data"].(map[string]interface{}); ok {
		if resultData, ok := data["result"].([]interface{}); ok && len(resultData) > 0 {
			if timestampVal, ok := resultData[0].(float64); ok {
				prometheusTime := time.Unix(int64(timestampVal), int64((timestampVal-float64(int64(timestampVal)))*1e9))
				localTime := time.Now()
				offset := localTime.Sub(prometheusTime)
				fmt.Printf("Local time: %s, Prometheus time: %s, offset: %v\n", localTime.Format(time.RFC3339), prometheusTime.Format(time.RFC3339), offset)
				return offset, nil
			}
		}
	}

	return 0, fmt.Errorf("could not extract timestamp from prometheus response")
}
