package domain

type Apps []struct {
	AppURL     string   `json:"app_url"`
	Method     string   `json:"method"`
	Recipients []string `json:"recipients"`
}
