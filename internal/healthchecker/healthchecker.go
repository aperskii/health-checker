package healthchecker

import (
	"encoding/json"
	"fmt"
	"git.ghpcard.local/csipitca/auth"
	"git.ghpcard.local/csipitca/email"
	"git.ghpcard.local/csipitca/logcs"
	"github.com/healthchecker/internal/domain"
	"net/url"
	"time"
)

var (
	jsonOrder = []string{"status", "info", "error", "details"}
)

type HealthChecker struct {
	authClient  *auth.ClientCredentialsClient
	emailClient *email.Client
	apps        []domain.App
}

func NewHealthChecker(authResourceClient *auth.ClientCredentialsClient, emailClient *email.Client, apps []domain.App) (*HealthChecker, error) {
	healthChecker := &HealthChecker{
		authClient:  authResourceClient,
		emailClient: emailClient,
		apps:        apps,
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
		<-tc.C
		u, err := url.Parse(app.AppURL)
		if err != nil {
			logcs.Error(err)
			continue
		}
		authResourceClient := self.authClient.NewResourceClient(fmt.Sprintf("%s://%s", u.Scheme, u.Hostname()))
		resp, err := authResourceClient.GET(u.Path)
		if err != nil {
			logcs.Error(err)
			continue
		}
		if resp.IsError() && resp.IsInternalServerError() {
			var jsonResponse map[string]interface{}
			err = json.Unmarshal(resp.Body(), &jsonResponse)
			if err != nil {
				logcs.Error(err)
				continue
			}
			if jsonResponse["status"] == "nok" || jsonResponse["status"] == "warn" {
				// create Map to store the health response
				var healthResponseMap = make(map[string]*HealthResponse)
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
											healthResponse, f := healthResponseMap[componentName]
											if !f {
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
				// initialise email to the recipients
				m := self.emailClient.NewHTMLMessage()
				for _, k := range app.Recipients {
					m.AddTo(k)
				}
				m.AddSubject(u.Host)
				m.AddHTMLBody("response.gohtml", healthResponseMap)
				if err := m.SendMessage(); err != nil {
					logcs.Error(err)
					continue
				}
			}
		}
	}
}
