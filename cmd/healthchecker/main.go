package main

import (
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

	healthC, err := healthchecker.NewHealthChecker(cfg.AuthClient.URL, cfg.AuthClient.ClientID, cfg.AuthClient.ClientSecret, cfg.AuthClient.GrantType)
	if err != nil {
		logcs.Error(err)
	}
	healthC.InitialiteMonitoring()
	//// Initiate email client
	//emailClient, err := email.NewClient(cfg.MailServer, logger)
	//if err != nil {
	//	logcs.Error(err)
	//}

	for {

	}
}
