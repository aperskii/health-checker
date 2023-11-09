package healthchecker

import (
	"encoding/json"
	"fmt"
	"git.ghpcard.local/csipitca/logcs"
	"gopkg.in/resty.v1"
	"net/http"
	"time"
)

type HealthChecker struct {
	client           *resty.Client
	authUrl          string
	authUserName     string
	authClientSecret string
	authGrantType    string
	token            string
}

type tokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
}

func NewHealthChecker(authUrl, authUserName, authClientSecret, authGrantType string) (*HealthChecker, error) {
	restyClient := resty.New()
	resp, err := restyClient.SetHostURL(authUrl).R().
		SetFormData(map[string]string{
			"client_id":     authUserName,
			"client_secret": authClientSecret,
			"grant_type":    authGrantType,
		}).
		Post("/token")
	if err != nil {
		logcs.Error(err)
		return nil, err
	}
	if resp.IsError() {
		logcs.Error(fmt.Errorf("Status Code: %d; Body: %s", resp.StatusCode(), string(resp.Body())))
		return nil, err
	}
	var response tokenResponse
	err = json.Unmarshal(resp.Body(), &response)
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
		token:            response.AccessToken,
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
	Status  string `json:"status"`
	Info    Info
	Error   Error
	Details struct {
		App struct {
			Arsh string
			Os   string
			Time time.Time
		}
		DiskUsage struct {
		}
	}
}
type Info struct {
	Name   string
	Status string
}
type Error struct {
	Name  string
	Error string
}

func (self *HealthChecker) check() error {
	client := resty.New()
	resp, err := client.SetHostURL("https://pinmanager-test.shell.com").R().
		SetHeader("Accept", "application/json").
		SetAuthToken(self.token).
		Get("/v1/health")
	if err != nil {
		logcs.Error(err)
		return err
	}
	if string(resp.Body()) == "access_token_expired" && resp.StatusCode() == 401 {
		fmt.Println("NEW ACCESS TOKEN")
		resp, err := client.SetHostURL(self.authUrl).R().
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
			logcs.Error(fmt.Errorf("Status Code: %d; Body: %s", resp.StatusCode(), string(resp.Body())))
			return nil
		}
		var response tokenResponse
		err = json.Unmarshal(resp.Body(), &response)
		if err != nil {
			logcs.Error(err)
			return err
		}
		self.token = response.AccessToken
		err = self.check()
		if err != nil {
			logcs.Error(err)
			return nil
		}
	}
	if resp.IsError() {
		if resp.StatusCode() == http.StatusInternalServerError {
			var x map[string]interface{}
			err = json.Unmarshal(resp.Body(), &x)
			if err != nil {
				logcs.Error(err)
				return err
			}
			for key, element := range x {
				if element == "nok" {
					fmt.Println(x["error"])
				}
			}
		}
	}
	return nil
}
