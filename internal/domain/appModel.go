package domain

import "time"

type App struct {
	AppURL               string        `json:"app_url"`
	Recipients           []string      `json:"recipients"`
	CheckIntervalSeconds time.Duration `json:"check_interval_seconds"`
	SendEmailSeconds     time.Duration `json:"send_email_seconds"`
}
