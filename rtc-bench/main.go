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
}

type Cluster struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type Iperf struct {
	Type           string     `mapstructure:"type"`
	Time           int        `mapstructure:"time"`
	Bandwidth      int        `mapstructure:"bandwidth"`
	Size           int        `mapstructure:"size"`
	Enhanced       bool       `mapstructure:"enhanced"`
	ReportInterval int        `mapstructure:"report_interval"`
	Connection     Connection `mapstructure:"connection"`
}

type Connection struct {
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

	// Actual values used during simple-tunnel testing
	//iperf -c localhost -p 5000 -u -i 1 -l 100 -b 800000 -t 0 -e
	//turncat --log=all:INFO udp://127.0.0.1:5000 k8s://stunner/udp-gateway:udp-listener udp://$(kubectl get svc iperf-server -o jsonpath="{.spec.clusterIP}"):5001

	// Configure iperf command
	var iperf []string
	iperf = append(iperf, "-c", cfg.Measurements[0].Iperf.Connection.Host)
	iperf = append(iperf, "-p", fmt.Sprintf("%d", cfg.Measurements[0].Iperf.Connection.Port))
	iperf = append(iperf, "-u")
	iperf = append(iperf, "-i", fmt.Sprintf("%d", cfg.Measurements[0].Iperf.ReportInterval))
	iperf = append(iperf, "-l", fmt.Sprintf("%d", cfg.Measurements[0].Iperf.Size))
	iperf = append(iperf, "-b", fmt.Sprintf("%d", cfg.Measurements[0].Iperf.Bandwidth))
	iperf = append(iperf, "-t", fmt.Sprintf("%d", cfg.Measurements[0].Iperf.Time))
	if cfg.Measurements[0].Iperf.Enhanced {
		iperf = append(iperf, "-e")
	}

	// Quick test to see if we can execute a command with the config values
	out, err := exec.Command("iperf", iperf...).Output()
	if err != nil {
		panic(fmt.Errorf("fatal error executing command: %w", err))
	}

	fmt.Printf("output is %s\n", (out))
}
