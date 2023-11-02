package status

import (
	"encoding/json"
	"fmt"
	"git.ghpcard.local/csipitca/logcs"
	"net/http"
	"time"
)

const (
	InvalidProfileId      = 105
	InvalidJson           = 301
	ImageNotBase64Encoded = 302
	InternalServerError   = 500
	StatusOk              = 200
	BadRequest            = 400
)

var (
	TIME_LAYOUT = "2006-01-02T15:04:05.999Z"
)

var statusText = map[int]string{
	InvalidProfileId:      "UID is missing",
	InvalidJson:           "Invalid JSON",
	ImageNotBase64Encoded: "Image data is not base64 encoded",
	InternalServerError:   "Internal Server Error",
	BadRequest:            "Bad Request",
}

func StatusText(code int) string {
	return statusText[code]
}

type BDMStatusError struct {
	Status      string `json:"Status"`
	Code        int    `json:"Code"`
	Description string `json:"Description"`
}

type StatusError struct {
	Type     string           `json:"type"`
	Title    string           `json:"title"`
	Status   int              `json:"status"`
	Detail   string           `json:"detail"`
	Messages []*StatusMessage `json:"messages"`
}

type StatusMessage struct {
	Code        int    `json:"code"`
	ID          string `json:"id"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
}

func BadRequestJSONResponse(w http.ResponseWriter, requestID string, statusMessages []*StatusMessage) {
	for _, statusMessage := range statusMessages {
		statusMessage.ID = requestID
		statusMessage.Type = "ERROR"
		statusMessage.Timestamp = time.Now().Format(TIME_LAYOUT)
	}

	statusError := &StatusError{
		Type:     "string",
		Title:    "Validation failure",
		Status:   BadRequest,
		Detail:   "There was an error during validation of the request",
		Messages: statusMessages,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(BadRequest)
	defer func() {
		if err := json.NewEncoder(w).Encode(statusError); err != nil {
			logcs.Error(fmt.Errorf("requestID: %s; %v", requestID, err))
		}
	}()
}

func StatusOkJSONResponse(w http.ResponseWriter, requestID string, statusMessages []*StatusMessage) {
	for _, statusMessage := range statusMessages {
		statusMessage.ID = requestID
		statusMessage.Type = "ERROR"
		statusMessage.Timestamp = time.Now().Format(TIME_LAYOUT)
	}

	statusError := &StatusError{
		Type:     "string",
		Title:    "Validation failure",
		Status:   StatusOk,
		Detail:   "There was an error during validation of the request",
		Messages: statusMessages,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(StatusOk)
	defer func() {
		if err := json.NewEncoder(w).Encode(statusError); err != nil {
			logcs.Error(fmt.Errorf("requestID: %s; %v", requestID, err))
		}
	}()
}

type InternalError struct {
	RequestID string `json:"request_id"`
}

func InternalServerJSONResponse(w http.ResponseWriter, requestID string) {
	statusError := &StatusError{
		Type:   "string",
		Title:  "Validation failure",
		Status: InternalServerError,
		Detail: "There was an error during validation of the request",
		Messages: []*StatusMessage{
			{
				Code:        InternalServerError,
				ID:          requestID,
				Description: StatusText(InternalServerError),
				Type:        "TECHNICAL_ERROR",
				Timestamp:   time.Now().Format(TIME_LAYOUT),
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(InternalServerError)
	defer func() {
		if err := json.NewEncoder(w).Encode(statusError); err != nil {
			logcs.Error(fmt.Errorf("requestID: %s; %v", requestID, err))
		}
	}()
}

func NotFoundHTMLResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func OKJSONResponse(w http.ResponseWriter, requestID string, any interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	defer func() {
		if err := json.NewEncoder(w).Encode(any); err != nil {
			logcs.Error(fmt.Errorf("requestID: %s; %v", requestID, err))
		}
	}()
}

func CreatedJSONResponse(w http.ResponseWriter, requestID string, any interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	defer func() {
		if err := json.NewEncoder(w).Encode(any); err != nil {
			logcs.Error(fmt.Errorf("requestID: %s; %v", requestID, err))
		}
	}()
}
