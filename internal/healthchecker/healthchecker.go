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

func (self *HealthChecker) InitialiteMonitoring() {
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
			}
			err = self.check()
			if err != nil {
				logcs.Error(err)
				return nil
			}
		}

		if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusInternalServerError {
			var x map[string]interface{}
			err = json.Unmarshal(resp.Body(), &x)
			//file, err := ioutil.ReadFile("./response.json")
			//err = json.Unmarshal(file, &x)
			if err != nil {
				logcs.Error(err)
				return err
			}
			if x["status"] == "nok" || x["status"] == "warn" {
				var healthResponseMap = make(map[string]*HealthResponse)
				for element, k := range x {
					if element == "info" {
						if rec, ok := k.(map[string]interface{}); ok {
							for s, t := range rec {
								if res, o := t.(map[string]interface{}); o {
									if res["status"] == "nok" || res["status"] == "warn" {
										hr, f := healthResponseMap[s]
										if !f {
											hr = &HealthResponse{}
										}
										hr = &HealthResponse{}
										healthResponseMap[s] = hr
										hr.Name = s
										hr.Status = fmt.Sprintf("%s", res["status"])
									}
								}
							}
						}
					}
					if element == "error" {
						if rec, ok := k.(map[string]interface{}); ok {
							for key, v := range rec {
								if _, ok := healthResponseMap[key]; ok {
									if key == healthResponseMap[key].Name {
										healthResponseMap[key].Error = fmt.Sprintf("%v", v)
									}
								}
							}
						}
					}
					if element == "details" {
						if rec, ok := k.(map[string]interface{}); ok {
							for key, v := range rec {
								if _, ok := healthResponseMap[key]; ok {
									if key == healthResponseMap[key].Name && v != nil {
										healthResponseMap[key].Details = map[string]interface{}{
											key: v,
										}
									}
								}
							}
						}
					}
				}
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
				//fmt.Println(healthResponseMap)
				//for k, v := range healthResponseMap {
				//	fmt.Printf("error is : %s\n", k)
				//	fmt.Println(v.Name)
				//	fmt.Println(v.Status)
				//	fmt.Println(v.Error)
				//	fmt.Println(v.Details)
				//}
			}
		}
	}
	return nil
}

// InitialiseClient Client with access token
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
