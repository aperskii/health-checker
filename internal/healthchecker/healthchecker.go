package healthchecker

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/url"
	"strings"
	"time"
)

var (
	jsonOrder = []string{"status", "info", "error", "details"}
)

type HealthChecker struct {
	authClient *auth.ClientCredentialsClient
	App        App
}

type App struct {
	AppName              string        `json:"app_name"`
	AppURL               string        `json:"app_url"`
	Recipients           []string      `json:"recipients"`
	CheckIntervalSeconds time.Duration `json:"check_interval_seconds"`
}

func NewHealthChecker(authResourceClient *auth.ClientCredentialsClient, app App) *HealthChecker {
	healthChecker := &HealthChecker{
		authClient: authResourceClient,
		App:        app,
	}
	return healthChecker
}

func (self *HealthChecker) InitialiseMonitoring() {
	self.check()
	// Wait for the interval before checking again
	time.Sleep(self.App.CheckIntervalSeconds * time.Second)
}

type HealthResponse struct {
	AppUrl  string
	Name    string
	Status  string
	Error   string
	Details map[string]interface{}
}

func (self *HealthChecker) check() {
	requestID := uuid.New().String()
	u, err := url.Parse(self.App.AppURL)
	if err != nil {
		logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
			"app-url": self.App.AppURL,
		}).Error(err)
		return
	}
	logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
		"app-name": self.App.AppName,
	}).Success("starting health check")
	authResourceClient := self.authClient.NewResourceClient(u.Scheme + "://" + u.Hostname())
	resp, err := authResourceClient.GET("", u.Path)
	if err != nil {
		logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
			"app-name": self.App.AppName,
		}).Error(err)
		return
	}
	if resp.IsBadRequest() {
		logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
			"app-name":    self.App.AppName,
			"status-code": resp.StatusCode(),
		}).Info("receive bad request")
		return
	}
	if resp.IsSuccess() {
		logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
			"app-name":    self.App.AppName,
			"status-code": resp.StatusCode(),
		}).Success("receive successfully request")
		return
	}
	if resp.IsError() && resp.IsInternalServerError() {
		logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
			"app-name":    self.App.AppName,
			"status-code": resp.StatusCode(),
		}).Info("receive server error")
		var jsonResponse map[string]interface{}
		err = json.Unmarshal(resp.Body(), &jsonResponse)
		if err != nil {
			logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
				"app-name":      self.App.AppName,
				"response-body": formatJson(resp.Body()),
			}).Error(err)
			return
		}
		if jsonResponse["status"] == "nok" || jsonResponse["status"] == "warn" {
			logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
				"app-name":        self.App.AppName,
				"app-url":         u.String(),
				"response-status": jsonResponse["status"],
			}).Info("response status")
			//create Map to store the health response
			healthResponse := new(HealthResponse)
			// iterate by the slice of string for ordered the map of the response json
			for _, orderedMap := range jsonOrder {
				// iterate by the json response
				for responseKey, responseValue := range jsonResponse {
					// ordered the map json to check first the status -> info -> error -> details
					if responseKey == orderedMap {
						if responseValueMap, ok := responseValue.(map[string]interface{}); ok {
							// range by Element info to check which component is not ok
							for componentName, componentInfo := range responseValueMap {
								if componentInfoMap, yes := componentInfo.(map[string]interface{}); yes {
									// check the value of status of the component if not ok or warn
									if componentInfoMap["status"] == "nok" || componentInfoMap["status"] == "warn" {
										logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
											"app-name":           self.App.AppName,
											"component_name":     componentName,
											"component_info_map": componentInfoMap,
										}).Info("component not ok")
										// put the name and status of component which is not ok in map
										healthResponse = &HealthResponse{
											AppUrl: self.App.AppName,
											Name:   componentName,
											Status: fmt.Sprintf("%s", componentInfoMap["status"]),
										}
									}
								}
								// check if there is an error for the component which is not ok
								if responseKey == "error" {
									logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
										"app-name":       self.App.AppName,
										"component_name": componentName,
										"component_info": componentInfo,
									}).Info("component error found")
									healthResponse.Error = fmt.Sprintf("%v", componentInfo)
								}
								// check if there is an details for the component which is not ok
								if responseKey == "details" {
									logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
										"app-name":       self.App.AppName,
										"component_name": componentName,
										"component_info": componentInfo,
									}).Info("component details found")
									healthResponse.Details = map[string]interface{}{
										componentName: componentInfo,
									}
								}
								if healthResponse.Error != "" {
									healthResponse.Error = "error not found"
								}
								if healthResponse.Details != nil {
									healthResponse.Details = map[string]interface{}{
										"Details": "not found",
									}
								}
								logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
									"app-url":           healthResponse.AppUrl,
									"component-name":    healthResponse.Name,
									"component-error":   healthResponse.Error,
									"component-details": healthResponse.Details,
								}).Error(fmt.Errorf("health check result"))
							}
						}
					}
				}
			}
			logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
				"app-name": self.App.AppName,
			}).Success("health check finished")
		} else {
			logcs.WithRequestId(requestID).WithAdditionalData(map[string]interface{}{
				"app-url":           u.String(),
				"status-code":       resp.StatusCode(),
				"components-status": jsonResponse["status"],
			}).Info("all components are healthy")
			return
		}
	}
}

func formatJson(s []byte) string {
	return strings.Replace(string(s), `\n`, ` `, -1)
}
