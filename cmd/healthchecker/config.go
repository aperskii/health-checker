package main

import (
	"encoding/json"
	"fmt"
	"git.ghpcard.local/csipitca/aes"
	"git.ghpcard.local/csipitca/email"
	"git.ghpcard.local/csipitca/hash"
	"git.ghpcard.local/csipitca/logcs"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"syscall"
)

type Config struct {
	AppName      string        `json:"app_name"`
	Env          string        `json:"env"`
	Version      string        `json:"version"`
	Log          *logcs.Config `json:"log"`
	HTTPSServer  HTTPSServer   `json:"https_server"`
	AuthResource AuthResource  `json:"auth_resource"`
	AuthClient   AuthClient    `json:"auth_client"`
	MailServer   *email.Config `json:"mail_server"`
}

type AuthClient struct {
	URL          string `json:"url"`
	LocalURL     string `json:"local_url"`
	Language     string `json:"language"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
}

type HTTPSServer struct {
	NetworkType string `json:"network_type"`
	Port        int    `json:"port"`
	PublicKey   string `json:"certificate_public_key"`
	PrivateKey  string `json:"certificate_private_key"`
}

type AuthResource struct {
	URL        string `json:"url"`
	Language   string `json:"language"`
	ResourceID string `json:"resource_id"`
}

func (c Config) isProd() bool {
	return c.Env == "prod"
}

func LoadConfig(authMethod, passwordFirstPart, passwordSecondPart, cfgFilePath string) (*Config, error) {
	password1 := []byte(passwordFirstPart)
	password2 := []byte(passwordSecondPart)
	if authMethod == LOAD_CONFIG_AUTH_METHOD_DUAL_SPLIT_CONSOLE {
		var err error
		//Read the dual split password to decrypt config file
		fmt.Print("Enter the first part of the password: ")
		password1, err = terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return nil, err
		}
		fmt.Println("Done!")
		fmt.Print("Enter the second part of the password: ")
		password2, err = terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return nil, err
		}
		fmt.Println("Done!")
	}

	f, err := os.Open(cfgFilePath)
	if err != nil {
		return nil, err
	}
	var c Config
	dec := json.NewDecoder(f)
	err = dec.Decode(&c)
	if err != nil {
		return nil, err
	}
	password, err := getPassword(password1, password2)
	if err != nil {
		return nil, err
	}
	return decryptConfig(&c, password)
}

func decryptConfig(c *Config, password string) (*Config, error) {
	aesClient, err := aes.NewStringCipher(password)
	if err != nil {
		return nil, err
	}

	authClientSecret, err := aesClient.Decrypt(c.AuthClient.ClientSecret)
	if err != nil {
		return nil, err
	}
	c.AuthClient.ClientSecret = authClientSecret

	if c.Log.LogErrEmail.Enabled {
		logEmailPass, err := aesClient.Decrypt(c.Log.LogErrEmail.EmailCfg.Password)
		if err != nil {
			return nil, err
		}
		c.Log.LogErrEmail.EmailCfg.Password = logEmailPass
	}

	return c, nil
}

func getPassword(password1, password2 []byte) (string, error) {
	concatinated := string(password1) + string(password2)
	if len(concatinated) > 32 {
		return concatinated[0:32], nil
	}
	if len(concatinated) < 32 {
		ha := hash.NewSHA512()
		hashed, err := ha.StdEncodedHASH(concatinated)
		if err != nil {
			return "", err
		}
		return concatinated + hashed[0:(32-len(concatinated))], nil
	}
	return concatinated, nil
}
