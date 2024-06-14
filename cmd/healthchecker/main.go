package main

import (
	"fmt"
	"git.ghpcard.local/csipitca/auth"
	"git.ghpcard.local/csipitca/logcs"
	"github.com/healthchecker/internal/healthchecker"
	"log"
	"os"
)

const (
	LOAD_CONFIG_AUTH_METHOD_DUAL_SPLIT         = "dual_split"
	LOAD_CONFIG_AUTH_METHOD_DUAL_SPLIT_CONSOLE = "dual_split_console"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("Specify config file path as argument.")
		os.Exit(1)
	}
	cfgFilePath := os.Args[1]

	cfg, err := LoadConfig(LOAD_CONFIG_AUTH_METHOD_DUAL_SPLIT, "TestDevEnvPasswo", "rd0123456789Test", cfgFilePath)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	err = logcs.InitLog(cfg.Log)
	if err != nil {
		logcs.Error(err)
		os.Exit(1)
	}

	logger, err := logcs.GetLogger()
	if err != nil {
		logcs.Error(err)
		os.Exit(1)
	}

	// Enqueue jobs
	for _, client := range cfg.AuthClient {
		go func(client AuthClient) {
			authClient, err := auth.NewClientCredentialsClient(client.URL, client.ClientID, client.ClientSecret, logger, false)
			if err != nil {
				logcs.WithAdditionalData(map[string]interface{}{
					"client_url": client.URL,
					"client_id":  client.ClientID,
				}).Error(err)
				return
			}
			logcs.WithAdditionalData(map[string]interface{}{
				"client_url": client.URL,
				"client_id":  client.ClientID,
			}).Success("client login successfully")

			healthC := healthchecker.NewHealthChecker(authClient, client.Application, client.NumWorkers)
			healthC.InitialiseMonitoring()
		}(client)
	}

	ch := make(chan int)
	<-ch
	fmt.Println()
}

//func connect(client AuthClient, logger *logcs.Logger) {
//	authClient, err := auth.NewClientCredentialsClient(client.URL, client.ClientID, client.ClientSecret, logger, false)
//	if err != nil {
//		logcs.WithAdditionalData(map[string]interface{}{
//			"client_url": client.URL,
//			"client_id":  client.ClientID,
//		}).Error(err)
//		return
//	}
//	logcs.WithAdditionalData(map[string]interface{}{
//		"client_url": client.URL,
//		"client_id":  client.ClientID,
//	}).Success("Client login successfully")
//	healthC := healthchecker.NewHealthChecker(authClient, client.Application)
//	healthC.InitialiseMonitoring()
//}
