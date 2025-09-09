package main

import (
	"a0/internal/app/server"
	"a0/internal/config"
	"a0/internal/logger"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {

	configPath := flag.String("config", "", "Path to the config file (YAML, JSON, TOML)")
	flag.Parse()

	if *configPath == "" {

		execPath, err := os.Executable()
		if err != nil {
			fmt.Println("Error: cannot determine executable path:", err)
			os.Exit(1)
		}
		execDir := filepath.Dir(execPath)

		wd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error: cannot get working directory:", err)
		}
		fmt.Println(execDir)

		dirs := []string{wd, execDir}
		candidates := []string{".env", "config.yaml", "config.json", "config.toml"}

		found := ""
		for _, dir := range dirs {
			for _, file := range candidates {
				fullPath := filepath.Join(dir, file)
				if _, err := os.Stat(fullPath); err == nil {
					found = fullPath
					break
				}
			}
			if found != "" {
				break
			}
		}

		if found == "" {
			fmt.Println("Error: no --config flag provided and no config file found in executable/working directory.")
			os.Exit(1)
		}
		*configPath = found
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		panic(err)
	}

	log, err := logger.NewZeroLogLoggerBuilder(false).
		WithLevel(cfg.Server.LogLevel).
		WithConsole(true).
		WithTimeFormat("02-01-2006 15:04:05").
		WithGlobal().
		WithCaller(true).
		Build()
	if err != nil {
		panic(err)
	}

	log.Info().Msg("zerolog configured.")

	server.StartServer(cfg, log)
}
