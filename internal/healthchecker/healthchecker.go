package healthchecker

import (
	"encoding/json"
	"fmt"
	"git.ghpcard.local/csipitca/auth"
	"git.ghpcard.local/csipitca/logcs"
	"github.com/healthchecker/internal/domain"
	"net/url"
	"time"
)

var (
	jsonOrder = []string{"status", "info", "error", "details"}
)

type HealthChecker struct {
	authClient *auth.ClientCredentialsClient
	apps       []domain.App
}

func NewHealthChecker(authResourceClient *auth.ClientCredentialsClient, apps []domain.App) (*HealthChecker, error) {
	healthChecker := &HealthChecker{
		authClient: authResourceClient,
		apps:       apps,
	}
	return healthChecker, nil
}

func (self *HealthChecker) InitialiseMonitoring() {
	go self.monitoring()
}

func (self *HealthChecker) monitoring() {
	for _, app := range self.apps {
		go self.check(app)
	}
}

type HealthResponse struct {
	AppUrl  string
	Name    string
	Status  string
	Error   string
	Details map[string]interface{}
}

func (self *HealthChecker) check(app domain.App) {
	for {
		tc := time.NewTicker(app.CheckIntervalSeconds * time.Second)
		//Start timer for 3 seconds
		MyTimer := time.NewTimer(app.SendEmailSeconds * time.Second)
		<-tc.C
		u, err := url.Parse(app.AppURL)
		if err != nil {
			logcs.Error(err)
			continue
		}
		authResourceClient := self.authClient.NewResourceClient(fmt.Sprintf("%s://%s", u.Scheme, u.Hostname()))
		resp, err := authResourceClient.GET("", u.Path)
		if err != nil {
			logcs.Error(err)
			continue
		}
		//create Map to store the health response
		var healthResponseMap = make(map[string]*HealthResponse)
		var previousResponce = make(map[string]*HealthResponse)
		if len(previousResponce) == 0 {
			previousResponce = healthResponseMap
		}
		//if resp.IsError() && resp.IsInternalServerError() {
		var jsonResponse map[string]interface{}
		err = json.Unmarshal(resp.Body(), &jsonResponse)
		if err != nil {
			logcs.Error(err)
			continue
		}
		if jsonResponse["status"] == "nok" || jsonResponse["status"] == "warn" {
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
										// initialise the map
										healthResponse, ok := healthResponseMap[componentName]
										if !ok {
											healthResponse = &HealthResponse{}
										}
										healthResponse = &HealthResponse{}
										healthResponseMap[componentName] = healthResponse
										// put the name and status of component which is not ok in map
										healthResponse.Name = componentName
										healthResponse.Status = fmt.Sprintf("%s", componentInfoMap["status"])
									}
								}
								// check if there is an error for the component which is not ok
								if _, ok := healthResponseMap[componentName]; ok && responseKey == "error" {
									healthResponseMap[componentName].Error = fmt.Sprintf("%v", componentInfo)
								}
								// check if there is an details for the component which is not ok
								if _, ok := healthResponseMap[componentName]; ok && responseKey == "details" {
									healthResponseMap[componentName].Details = map[string]interface{}{
										componentName: componentInfo,
									}
								}
							}
						}
					}
				}
			}
			// testing if the last map not equal a new map response
			if ok := compareMaps(healthResponseMap, previousResponce); ok {
				formatEmail(healthResponseMap)
				continue
			} else {
				select {
				case <-MyTimer.C:
					formatEmail(healthResponseMap)
					break
				default:
					continue
				}
				<-time.After(app.SendEmailSeconds * time.Second)
			}
			previousResponce = healthResponseMap
			fmt.Println("map after ", previousResponce)
		}
	}
}

// Compare the Error in the two Map, and return true when not equal
func compareMaps(nextMap, prevMap map[string]*HealthResponse) bool {
	for i, v := range nextMap {
		if v.Error != prevMap[i].Error {
			return true
		}
	}
	return false
}

// formatting email and send it with logcs
func formatEmail(healthResponseMap map[string]*HealthResponse) {
	for _, value := range healthResponseMap {
		if len(value.Details) == 0 {
			value.Details = map[string]interface{}{
				"Details": "not found",
			}
		}
		err := fmt.Errorf("App name is : %s\n, Name : %s\n, Error : %s\n, Details is : %s\n", value.AppUrl, value.Name, value.Error, value.Details)
		logcs.Error(err)
	}
}
