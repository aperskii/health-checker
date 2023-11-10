package healthchecker

import (
	"encoding/json"
	"fmt"
	"git.ghpcard.local/csipitca/email"
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
	emailClient      *email.Client
}

type tokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
}

func NewHealthChecker(authUrl, authUserName, authClientSecret, authGrantType string, emailClient *email.Client) (*HealthChecker, error) {
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
		emailClient:      emailClient,
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
	Info    Info
	Error   Error
	Details interface{}
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

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusInternalServerError {

		var x map[string]interface{}
		err = json.Unmarshal(resp.Body(), &x)
		if err != nil {
			logcs.Error(err)
			return err
		}
		if x["status"] == "nok" || x["status"] == "warm" {
			var hr HealthResponse
			for e, k := range x {
				if e == "info" {
					if rec, ok := k.(map[string]interface{}); ok {
						for s, t := range rec {
							if res, ok := t.(map[string]interface{}); ok {
								for _, v := range res {
									if v == "nok" || v == "warn" {
										hr.Info.Name = s
										hr.Info.Status = v.(string)
									}
									//	fmt.Println(s, "value is", v)
								}
							}
						}
					}
				}
				if e == "error" {
					if rec, ok := k.(map[string]interface{}); ok {
						for e, _ := range rec {
							result, ok := rec[hr.Info.Name]
							if ok {
								hr.Error.Name = e
								hr.Error.Error = result.(string)
								fmt.Println(hr.Error.Error)
							}
						}
					}
				}
				if e == "details" {
					if rec, ok := k.(map[string]interface{}); ok {
						for e, v := range rec {
							hr.Details = "Details not found"
							if e == hr.Info.Name {
								hr.Details = v
							}
						}
					}
				}
			}
			m := self.emailClient.NewHTMLMessage()
			m.AddTo("yberqiqch@tc-s.eu")
			m.AddSubject("test HealthChecker")
			m.AddHTMLBody("response.gohtml", hr)
			if err := m.SendMessage(); err != nil {
				logcs.Error(err)
				return err
			}

			fmt.Println(hr.Info.Name)
			fmt.Println(hr.Info.Status)
			fmt.Println(hr.Error.Name)
			fmt.Println(hr.Error.Error)
			fmt.Println(hr.Details)
		}
	}
	//if resp.IsError() {
	//	if resp.StatusCode() == http.StatusInternalServerError {
	//		var x map[string]interface{}
	//		err = json.Unmarshal(resp.Body(), &x)
	//		if err != nil {
	//			logcs.Error(err)
	//			return err
	//		}
	//		//for _, element := range x {
	//		//	//if _, ok := element.(map[string]string); ok {
	//		//	//	fmt.Println(x["info"])
	//		//	//}
	//		//}
	//		fmt.Println(x["info"])
	//		//fmt.Println(x)
	//	}
	//}
	return nil
}
