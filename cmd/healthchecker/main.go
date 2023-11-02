package main

import (
	"crypto/tls"
	"fmt"
	"git.ghpcard.local/csipitca/auth"
	"git.ghpcard.local/csipitca/logcs"
	"github.com/gorilla/mux"
	"github.com/healthchecker/internal/middleware"
	"github.com/healthchecker/internal/services"
	"log"
	"net"
	"net/http"
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

	servs, err := services.NewServices(
		services.WithRequest(),
	)
	if err != nil {
		logcs.Error(err)
		os.Exit(1)
	}

	authResource, err := auth.NewResource(cfg.AuthResource.URL, cfg.AuthResource.Language, cfg.AuthResource.ResourceID, logger, false)
	if err != nil {
		logcs.Error(err)
		os.Exit(1)
	}

	authClient, err := auth.NewClientCredentialsClient(cfg.AuthClient.URL, cfg.AuthClient.ClientID, cfg.AuthClient.ClientSecret, logger, false)
	if err != nil {
		logcs.Error(err)
		os.Exit(1)
	}

	ridMw := &middleware.RequestID{}

	r := mux.NewRouter()
	r.StrictSlash(true)
	r.HandleFunc("/fotoscan/v1/ping", ridMw.Set(authResource.Secure("v_one_ping", ctrls.AppController.GetPing))).Methods("GET")
	r.HandleFunc("/fotoscan/v1/validierung/auftrag", ridMw.Set(authResource.Secure("v_one_request", ctrls.AppController.PostValidateOrder))).Methods("POST")
	r.HandleFunc("/api/digital/{uid}", ridMw.Set(ctrls.AppController.ReceiveResponseDigital)).Methods("GET")
	r.HandleFunc("/api/analogue/{uid}", ridMw.Set(ctrls.AppController.ReceiveResponseAnalogue)).Methods("GET")

	tlsCfg := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		CipherSuites: []uint16{
			// TLS 1.2 cipher suites
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,

			// TLS 1.3 cipher suites
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
		},
	}
	server := &http.Server{
		Handler:      r,
		TLSConfig:    tlsCfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	fmt.Printf("Starting the server on :%v\n", cfg.HTTPSServer.Port)
	l, err := net.Listen("tcp4", fmt.Sprintf(":%d", cfg.HTTPSServer.Port))
	if err != nil {
		logcs.Error(err)
	}
	err = server.ServeTLS(l, cfg.HTTPSServer.PublicKey, cfg.HTTPSServer.PrivateKey)
	if err != nil {
		logcs.Error(err)
	}
}
