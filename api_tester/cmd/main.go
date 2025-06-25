package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/belalakhter/packages/api_tester/internal/flv"
	"github.com/belalakhter/packages/api_tester/internal/hls"
	"github.com/belalakhter/packages/api_tester/internal/sse"
	"github.com/belalakhter/packages/api_tester/internal/ws"
	"github.com/belalakhter/packages/api_tester/utils"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Addr         string `yaml:"addr"`
	InitialCount int64  `yaml:"initial_count"`
	Duration     int64  `yaml:"duration"`
	PumpCount    int64  `yaml:"pump_count"`
	Type         string `yaml:"type"`
}

func loadConfig(configPath string) (*Config, error) {
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", absPath)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}

	if config.Addr == "" {
		return nil, fmt.Errorf("addr is required in config")
	}
	if config.Type == "" {
		return nil, fmt.Errorf("type is required in config")
	}
	if config.InitialCount <= 0 {
		return nil, fmt.Errorf("initial_count must be greater than 0")
	}
	if config.Duration <= 0 {
		return nil, fmt.Errorf("duration must be greater than 0")
	}
	if config.PumpCount <= 0 {
		return nil, fmt.Errorf("pump_count must be greater than 0")
	}

	return &config, nil
}

func main() {
	if len(os.Args) < 2 {
		utils.LogMessage("Usage: ./api_tester <config.yaml>", utils.Fatal_Error_Code)
		return
	}

	configPath := os.Args[1]

	config, err := loadConfig(configPath)
	if err != nil {
		utils.LogMessage(fmt.Sprintf("Error loading config: %v", err), utils.Fatal_Error_Code)
		return
	}

	ctx := context.Background()

	switch config.Type {
	case "ws":
		ws.RunWebsocketTest(ctx, config.Addr, config.InitialCount, config.PumpCount, config.Duration)
	case "sse":
		sse.RunSseTest(ctx, config.Addr, config.InitialCount, config.PumpCount, config.Duration)
	case "hls":
		hls.RunHlsTest(ctx, config.Addr, config.InitialCount, config.PumpCount, config.Duration)
	case "flv":
		flv.RunFlvTest(ctx, config.Addr, config.InitialCount, config.PumpCount, config.Duration)
	default:
		utils.LogMessage(fmt.Sprintf("Unknown connection type: %s. Supported types: ws, sse, hls, flv", config.Type), utils.Fatal_Error_Code)
	}
}
