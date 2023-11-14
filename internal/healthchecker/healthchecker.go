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
	apps             domain.Apps
}

type TokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
}

func NewHealthChecker(authUrl, authUserName, authClientSecret, authGrantType string, emailClient *email.Client, apps domain.Apps) (*HealthChecker, error) {
	restyClient := resty.New()
	accessToken, err := GenerateToken(restyClient, authUrl, authUserName, authClientSecret, authGrantType)
	if err != nil {
		logcs.Error(err)
		return nil, err
	}
	return &HealthChecker{
		client:           restyClient,
		authUrl:          authUrl,
		authUserName:     authUserName,
		authClientSecret: authClientSecret,
		authGrantType:    authGrantType,
		token:            accessToken,
		emailClient:      emailClient,
		apps:             apps,
	}, nil
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
	AppUrl  string
	Info    Info
	Error   Error
	Details map[string]interface{}
}

type Info struct {
	Name   string
	Status string
}
type Error struct {
	Name string
	Info string
}

func (self *HealthChecker) check() error {
	client := resty.New()
	for _, p := range self.apps {
		u, err := url.Parse(p.AppURL)
		if err != nil {
			logcs.Error(err)
		}
		resp, err := client.SetHostURL(fmt.Sprintf("%s://%s", u.Scheme, u.Hostname())).R().
			SetHeader("Accept", "application/json").
			SetAuthToken(self.token).
			Get(u.RequestURI())
		if err != nil {
			logcs.Error(err)
			return err
		}
		if string(resp.Body()) == "access_token_expired" && resp.StatusCode() == 401 {
			newToken, err := GenerateToken(client, self.authUrl, self.authUserName, self.authClientSecret, self.authGrantType)
			if err != nil {
				logcs.Error(err)
			}
			self.token = newToken
			err = self.check()
			if err != nil {
				logcs.Error(err)
				return nil
			}
		}

		if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusInternalServerError {
			var x map[string]interface{}
			err = json.Unmarshal(resp.Body(), &x)
			if err != nil {
				logcs.Error(err)
				return err
			}
			if x["status"] == "nok" || x["status"] == "warn" {
				var hr HealthResponse
				hr.AppUrl = p.AppURL
				for e, k := range x {
					if e == "info" {
						if rec, ok := k.(map[string]interface{}); ok {
							for s, t := range rec {
								if res, o := t.(map[string]interface{}); o {
									for _, v := range res {
										if v == "nok" || v == "warn" {
											hr.Info.Name = s
											hr.Info.Status = v.(string)
										}
									}
								}
							}
						}
					}
					if e == "error" {
						if rec, ok := k.(map[string]interface{}); ok {
							for key, v := range rec {
								if key == hr.Info.Name {
									hr.Error.Name = key
									hr.Error.Info = v.(string)
								}
							}
						}
					}
					if e == "details" {
						if rec, ok := k.(map[string]interface{}); ok {
							for key, v := range rec {
								if key == hr.Info.Name {
									hr.Details = map[string]interface{}{
										key: v,
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
				m.AddHTMLBody("response.gohtml", hr)
				if err := m.SendMessage(); err != nil {
					logcs.Error(err)
					return err
				}
			}
		}
	}
	return nil
}

// InitialiseClient Client with access token
func GenerateToken(client *resty.Client, authUrl, authUserName, authClientSecret, authGrantType string) (string, error) {
	resp, err := client.SetHostURL(authUrl).R().
		SetFormData(map[string]string{
			"client_id":     authUserName,
			"client_secret": authClientSecret,
			"grant_type":    authGrantType,
		}).
		Post("/token")
	if err != nil {
		logcs.Error(err)
		return "", err
	}
	if resp.IsError() {
		logcs.Error(fmt.Errorf("status Code: %d; Body: %s", resp.StatusCode(), string(resp.Body())))
		return "", err
	}
	var response TokenResponse
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		logcs.Error(err)
		return "", err
	}
	return response.AccessToken, nil
}
