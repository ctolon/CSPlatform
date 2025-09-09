package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"v0/internal/app/providers"
	"v0/internal/config"
	"v0/internal/logger"
	"v0/internal/server"
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
	configProviders := providers.NewConfigProvider(providers.LOCAL, "", providers.DOTENV, "", "")
	cfg := config.LoadConfigIntoStruct(configProviders)

	log, err := logger.NewZeroLogLoggerBuilder(false).
		WithLevel(cfg.LogLevel).
		WithConsole(true).
		WithTimeFormat("02-01-2006 15:04:05").
		WithGlobal().
		WithCaller(true).
		Build()
	if err != nil {
		panic(err)
	}

	log.Info().Msg("zerolog configured.")
	log.Info().Msgf("load config from: %s", *configPath)
	log.Info().Msgf("code server base host: %s", cfg.CodeServerBaseHost)
	log.Info().Msgf("code server base port: %d", cfg.CodeServerBasePort)
	log.Info().Msgf("code server with username: %v", cfg.CodeServerWithUsername)

	//log.Info().Msgf("%v", cfg)
	server.StartServer(cfg, log)
}
