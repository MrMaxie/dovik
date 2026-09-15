package tui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

const (
	ttyglassDiagnosticsURL   = "TTYGLASS_DIAGNOSTICS_URL"
	ttyglassDiagnosticsToken = "TTYGLASS_DIAGNOSTICS_TOKEN"
)

type devLogRecord struct {
	Time   time.Time      `json:"time"`
	Source string         `json:"source"`
	Level  string         `json:"level"`
	Event  string         `json:"event"`
	Fields map[string]any `json:"fields,omitempty"`
}

var devLogHTTPClient = &http.Client{Timeout: time.Second}

func devLog(event string, values ...any) {
	endpoint := os.Getenv(ttyglassDiagnosticsURL)
	token := os.Getenv(ttyglassDiagnosticsToken)
	if endpoint == "" || token == "" {
		return
	}
	fields := make(map[string]any, len(values)/2)
	for index := 0; index+1 < len(values); index += 2 {
		key, ok := values[index].(string)
		if ok {
			fields[key] = values[index+1]
		}
	}
	body, err := json.Marshal(devLogRecord{Time: time.Now().UTC(), Source: "dovik-tui", Level: "info", Event: event, Fields: fields})
	if err != nil {
		return
	}
	go func() {
		request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return
		}
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Content-Type", "application/json")
		response, err := devLogHTTPClient.Do(request)
		if err == nil {
			_ = response.Body.Close()
		}
	}()
}

func closeDevLog() {}
