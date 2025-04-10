package main

import (
	"fmt"
	"git.ghpcard.local/csipitca/auth"
	"git.ghpcard.local/csipitca/logcs"
	"github.com/healthchecker/internal/healthchecker"
	"log"
	"os"
	"sync"
	"time"
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

	// WaitGroup for all goroutines to finish
	var wg sync.WaitGroup

	// Channel to collect all applications from all users
	applications := make(chan *healthchecker.HealthChecker) // Adjust buffer size as needed

	// Start a goroutine to continuously add jobs
	for _, client := range cfg.AuthClient {
		wg.Add(1)
		go func(client AuthClient) {
			defer wg.Done()
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

			// Add applications to the channel
			for _, app := range client.Application {
				hc := healthchecker.NewHealthChecker(authClient, app)
				applications <- hc
			}
		}(client)
	}

	// Close the channel once all applications are added
	go func() {
		wg.Wait()
		close(applications)
	}()

	// Start worker pool
	numWorkers := cfg.NumWorkers
	if numWorkers == 0 {
		numWorkers = 1
	}
	jobs := make(chan *healthchecker.HealthChecker, numWorkers)

	// Start workers
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, jobs, &wg)
	}

	// Distribute jobs to workers
	go func() {
		for app := range applications {
			jobs <- app
		}
	}()
	// Wait for workers to finish
	wg.Wait()
	ch := make(chan int)
	<-ch
	fmt.Println()
}

func worker(id int, jobs chan *healthchecker.HealthChecker, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		log.Println("Worker", id, "started job for application", job.App.AppName)
		job.InitialiseMonitoring()
		log.Println("Worker", id, "finished job for application", job.App.AppName)
		// Sleep for the interval before re-enqueuing the application
		go func(job *healthchecker.HealthChecker) {
			time.Sleep(job.App.CheckIntervalSeconds * time.Second)
			jobs <- job
		}(job)
	}
}
