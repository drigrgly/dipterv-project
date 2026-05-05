package main

import (
	"fmt"
	"os/exec"

	"github.com/spf13/viper"
)

type Config struct {
	Measurements []Measurement `mapstructure:"measurements"`
}

type Measurement struct {
	Name    string  `mapstructure:"name"`
	Cluster Cluster `mapstructure:"cluster"`
	Iperf   Iperf   `mapstructure:"iperf"`
	Turncat Turncat `mapstructure:"turncat"`
}

type Cluster struct {
	Host string `mapstructure:"host"`
	Type string `mapstructure:"type"`
}

type Iperf struct {
	Type           string       `mapstructure:"type"`
	Time           int          `mapstructure:"time"`
	Bandwidth      int          `mapstructure:"bandwidth"`
	Size           int          `mapstructure:"size"`
	Enhanced       bool         `mapstructure:"enhanced"`
	ReportInterval int          `mapstructure:"report_interval"`
	Host           string       `mapstructure:"host"`
	Client         ClientServer `mapstructure:"client"`
}

type Turncat struct {
	Log               string `mapstructure:"log"`
	ClientAddress     string `mapstructure:"client-address"`
	TurnServerAddress string `mapstructure:"turn-server-address"`
	PeerHostAddress   string `mapstructure:"peer-host-address"`
}

type ClientServer struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

func main() {

	// Set up viper to read the config file
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	var cfg Config

	// Find and read the config file
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	// Unmarshal the config into the struct
	err = viper.Unmarshal(&cfg)
	if err != nil {
		panic(fmt.Errorf("fatal error unmarshaling file: %w", err))
	}

	//  loop through the measurements and print the values
	for _, measurement := range cfg.Measurements {

		peerHostAddress := "udp//"
		turnServerAddress := "turn://"

		if measurement.Cluster.Type == "local" {

			// Turncat
			// Turn server IP and port
			turnIp, err := exec.Command("kubectl", "get", "svc", "-n", "stunner", "udp-gateway", "-o", "jsonpath={.status.loadBalancer.ingress[0].ip}").Output()
			if err != nil {
				panic(fmt.Errorf("fatal error executing command: %w", err))
			}

			turnServerAddress += fmt.Sprintf("%s:", turnIp)

			turnPort, err := exec.Command("kubectl", "get", "gateway", "udp-gateway", "-n", "stunner", "-o", "jsonpath={.spec.listeners[?(@.name=='udp-listener')].port}").Output()
			if err != nil {
				panic(fmt.Errorf("fatal error executing command: %w", err))
			}

			turnServerAddress += fmt.Sprintf("%s?transport=udp", turnPort)

			// Iperf
			peerIp, err := exec.Command("kubectl", "get", "svc", "iperf-server", "-o", "jsonpath={.spec.clusterIP}").Output()
			if err != nil {
				panic(fmt.Errorf("fatal error executing command: %w", err))
			}
			peerHostAddress += fmt.Sprintf("%s:5001", peerIp)
		}

		measurement.Turncat.PeerHostAddress = peerHostAddress
		measurement.Turncat.TurnServerAddress = turnServerAddress

		fmt.Printf("Measurement Name: %s\n", measurement.Name)
		fmt.Printf("Measurement type: %s\n", measurement.Cluster.Type)
		fmt.Printf("Cluster IP: %s\n", measurement.Cluster.Host)
		fmt.Printf("Peer Address: %s\n", measurement.Turncat.PeerHostAddress)
		fmt.Printf("Turn server address: %s\n", measurement.Turncat.TurnServerAddress)
	}

	// Actual values used during simple-tunnel testing
	//iperf -c localhost -p 5000 -u -i 1 -l 100 -b 800000 -t 0 -e
	//turncat --log=all:INFO udp://127.0.0.1:5000 k8s://stunner/udp-gateway:udp-listener udp://$(kubectl get svc iperf-server -o jsonpath="{.spec.clusterIP}"):5001

	// Configure iperf command
	// var iperf []string
	// iperf = append(iperf, "-c", cfg.Measurements[0].Iperf.Client.Host)
	// iperf = append(iperf, "-p", fmt.Sprintf("%d", cfg.Measurements[0].Iperf.Client.Port))
	// iperf = append(iperf, "-u")
	// iperf = append(iperf, "-i", fmt.Sprintf("%d", cfg.Measurements[0].Iperf.ReportInterval))
	// iperf = append(iperf, "-l", fmt.Sprintf("%d", cfg.Measurements[0].Iperf.Size))
	// iperf = append(iperf, "-b", fmt.Sprintf("%d", cfg.Measurements[0].Iperf.Bandwidth))
	// iperf = append(iperf, "-t", fmt.Sprintf("%d", cfg.Measurements[0].Iperf.Time))
	// if cfg.Measurements[0].Iperf.Enhanced {
	// 	iperf = append(iperf, "-e")
	// }

	// Quick test to see if we can execute a command with the config values
	// out, err := exec.Command("iperf", iperf...).Output()
	// if err != nil {
	// 	panic(fmt.Errorf("fatal error executing command: %w", err))
	// }

	// fmt.Printf("output is %s\n", (out))
}
