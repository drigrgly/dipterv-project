package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

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
	Cluster       Cluster       `mapstructure:"cluster"`
	Client        ClientServer  `mapstructure:"client"`
	LoadGenerator LoadGenerator `mapstructure:"loadGenerator"`
	Turncat       Turncat       `mapstructure:"turncat"`
}

type ClientServer struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type Cluster struct {
	Host string `mapstructure:"host"`
	Type string `mapstructure:"type"`
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
	fmt.Printf("Measurement type: %s\n", measurement.Cluster.Type)
	fmt.Printf("Hosted Client address: %s:%d\n", measurement.Client.Host, measurement.Client.Port)
	fmt.Printf("Cluster IP: %s\n", measurement.Cluster.Host)
	fmt.Printf("Peer Address: %s\n", measurement.Turncat.PeerHostAddress)
	fmt.Printf("Turn server address: %s\n", measurement.Turncat.TurnServerAddress)
	fmt.Print("------------------------------\n")

	fmt.Print("Starting turncat\n")

	// Get the authentication information for turncat

	turncat := exec.Command("turncat", "--log=all:INFO", clientAddress, measurement.Turncat.TurnServerAddress, measurement.Turncat.PeerHostAddress)
	turncat.Stdout = os.Stdout

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

	fmt.Printf("Shutting down turncat for measurement: %s\n", measurement.Name)
	turncat.Process.Kill()

	fmt.Printf("Measurement completed: %s\n", measurement.Name)
}
