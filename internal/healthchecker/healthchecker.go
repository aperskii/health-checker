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
	tc := time.NewTicker(15 * time.Second)
	for {
		<-tc.C
		for _, app := range self.apps {
			app := app
			go func() {
				err := self.check(app)
				if err != nil {
					logcs.Error(err)
				}
			}()
		}
	}
}

type HealthResponse struct {
	AppUrl  string
	Name    string
	Status  string
	Error   string
	Details map[string]interface{}
}

func (self *HealthChecker) check(app domain.App) error {
	for {
		tc := time.NewTicker(app.CheckIntervalSeconds * time.Second)
		<-tc.C
		u, err := url.Parse(app.AppURL)
		if err != nil {
			logcs.Error(err)
			return err
		}
		authResourceClient := self.authClient.NewResourceClient(fmt.Sprintf("%s://%s", u.Scheme, u.Hostname()))
		resp, err := authResourceClient.GET(u.Path)
		if err != nil {
			logcs.Error(err)
			return err
		}
		if resp.IsError() && resp.IsInternalServerError() {
			var jsonResponse map[string]interface{}
			err = json.Unmarshal(resp.Body(), &jsonResponse)
			if err != nil {
				logcs.Error(err)
				return err
			}
			if jsonResponse["status"] == "nok" || jsonResponse["status"] == "warn" {
				// create Map to store the health response
				var healthResponseMap = make(map[string]*HealthResponse)
				// iterate by the slice of string for ordered the map of the response json
				for _, sortElementJson := range jsonOrder {
					// iterate by the json response
					for keyJsonResponce, valueJsonResponce := range jsonResponse {
						// ordered the map json to check first the status -> info -> error -> details
						if keyJsonResponce == sortElementJson {
							if valueJsonResponceIsMap, ok := valueJsonResponce.(map[string]interface{}); ok {
								// range by Element info to check which component is not ok
								for keyMapValueJsonResponce, valueMapValueJsonResponce := range valueJsonResponceIsMap {
									if valueMapValueJsonResponceIsMap, yes := valueMapValueJsonResponce.(map[string]interface{}); yes {
										// check the value of status of the component if not ok or warn
										if valueMapValueJsonResponceIsMap["status"] == "nok" || valueMapValueJsonResponceIsMap["status"] == "warn" {
											// initialise the map
											healthResponse, f := healthResponseMap[keyMapValueJsonResponce]
											if !f {
												healthResponse = &HealthResponse{}
											}
											healthResponse = &HealthResponse{}
											healthResponseMap[keyMapValueJsonResponce] = healthResponse
											// put the name and status of component which is not ok in map
											healthResponse.Name = keyMapValueJsonResponce
											healthResponse.Status = fmt.Sprintf("%s", valueMapValueJsonResponceIsMap["status"])
										}
									}
									// check if there is an error for the component which is not ok
									if _, ok := healthResponseMap[keyMapValueJsonResponce]; ok && keyJsonResponce == "error" {
										healthResponseMap[keyMapValueJsonResponce].Error = fmt.Sprintf("%v", valueMapValueJsonResponce)
									}
									// check if there is an details for the component which is not ok
									if _, ok := healthResponseMap[keyMapValueJsonResponce]; ok && keyJsonResponce == "details" {
										healthResponseMap[keyMapValueJsonResponce].Details = map[string]interface{}{
											keyMapValueJsonResponce: valueMapValueJsonResponce,
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
					return err
				}
			}
		}
		return nil
	}
}
