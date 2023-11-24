package healthchecker

import (
	"encoding/json"
	"fmt"
	"git.ghpcard.local/csipitca/email"
	"git.ghpcard.local/csipitca/logcs"
	"github.com/healthchecker/internal/domain"
	"gopkg.in/resty.v1"
	"net/http"
	"net/url"
	"time"
)

var (
	jsonOrder = []string{"status", "info", "error", "details"}
)

type HealthChecker struct {
	client           *resty.Client
	authUrl          string
	authUserName     string
	authClientSecret string
	authGrantType    string
	token            string
	emailClient      *email.Client
	apps             []domain.App
}

type TokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
}

func NewHealthChecker(authUrl, authUserName, authClientSecret, authGrantType string, emailClient *email.Client, apps []domain.App) (*HealthChecker, error) {
	restyClient := resty.New()
	healthChecker := &HealthChecker{
		client:           restyClient,
		authUrl:          authUrl,
		authUserName:     authUserName,
		authClientSecret: authClientSecret,
		authGrantType:    authGrantType,
		emailClient:      emailClient,
		apps:             apps,
	}
	err := healthChecker.GenerateToken()
	if err != nil {
		logcs.Error(err)
		return nil, err
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
		err := self.check()
		if err != nil {
			logcs.Error(err)
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

func (self *HealthChecker) check() error {
	for _, p := range self.apps {
		u, err := url.Parse(p.AppURL)
		if err != nil {
			logcs.Error(err)
			return err
		}
		resp, err := self.client.SetHostURL(fmt.Sprintf("%s://%s", u.Scheme, u.Hostname())).R().
			SetHeader("Accept", "application/json").
			SetAuthToken(self.token).
			Get(u.RequestURI())
		if err != nil {
			logcs.Error(err)
			return err
		}
		if string(resp.Body()) == "access_token_expired" && resp.StatusCode() == 401 {
			err := self.GenerateToken()
			if err != nil {
				logcs.Error(err)
				return err
			}
			err = self.check()
			if err != nil {
				logcs.Error(err)
				return err
			}
		}
		if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusInternalServerError {
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
				for _, sortElement := range jsonOrder {
					// iterate by the json response
					for element, key := range jsonResponse {
						// ordered the map json to check first the status -> info -> error -> details
						if element == sortElement {
							if keyIsMap, ok := key.(map[string]interface{}); ok {
								// range by Element info to check which component is not ok
								for elem, value := range keyIsMap {
									if valueIsMap, yes := value.(map[string]interface{}); yes {
										// check the value of status of the component if not ok or warn
										if valueIsMap["status"] == "nok" || valueIsMap["status"] == "warn" {
											// initialise the map
											healthResponse, f := healthResponseMap[elem]
											if !f {
												healthResponse = &HealthResponse{}
											}
											healthResponse = &HealthResponse{}
											healthResponseMap[elem] = healthResponse
											// put the name and status of component which is not ok in map
											healthResponse.Name = elem
											healthResponse.Status = fmt.Sprintf("%s", valueIsMap["status"])
										}
									}
									// check if there is an error for the component which is not ok
									if _, ok := healthResponseMap[elem]; ok && element == "error" {
										healthResponseMap[elem].Error = fmt.Sprintf("%v", value)
									}
									// check if there is an details for the component which is not ok
									if _, ok := healthResponseMap[elem]; ok && element == "details" {
										healthResponseMap[elem].Details = map[string]interface{}{
											elem: value,
										}
									}
								}
							}
						}
					}
				}
				// initialise email to the recipients
				m := self.emailClient.NewHTMLMessage()
				for _, k := range p.Recipients {
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
	}
	return nil
}

// GenerateToken InitialiseClient Client with access token
func (self *HealthChecker) GenerateToken() error {
	resp, err := self.client.SetHostURL(self.authUrl).R().
		SetFormData(map[string]string{
			"client_id":     self.authUserName,
			"client_secret": self.authClientSecret,
			"grant_type":    self.authGrantType,
		}).
		Post("/token")
	if err != nil {
		logcs.Error(err)
		return err
	}
	if resp.IsError() {
		logcs.Error(fmt.Errorf("status Code: %d; Body: %s", resp.StatusCode(), string(resp.Body())))
		return err
	}
	var response TokenResponse
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		logcs.Error(err)
		return err
	}
	self.token = response.AccessToken
	return nil
}
