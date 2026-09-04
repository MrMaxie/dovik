//go:build dovik_dev_harness

package tui

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

const devLogPipeEnvironment = "DOVIK_TUI_DEV_LOG_PIPE"

type devLogRecord struct {
	Time   time.Time      `json:"time"`
	Source string         `json:"source"`
	Level  string         `json:"level"`
	Event  string         `json:"event"`
	Fields map[string]any `json:"fields,omitempty"`
}

type devLogClient struct {
	mutex   sync.Mutex
	closer  io.Closer
	encoder *json.Encoder
}

var (
	devLogOnce   sync.Once
	devLogTarget *devLogClient
)

func devLog(event string, values ...any) {
	devLogOnce.Do(func() {
		endpoint := os.Getenv(devLogPipeEnvironment)
		if endpoint == "" {
			return
		}
		connection, err := dialDevLog(endpoint)
		if err != nil {
			return
		}
		devLogTarget = &devLogClient{closer: connection, encoder: json.NewEncoder(connection)}
	})
	if devLogTarget == nil {
		return
	}
	fields := make(map[string]any, len(values)/2)
	for index := 0; index+1 < len(values); index += 2 {
		key, ok := values[index].(string)
		if ok {
			fields[key] = values[index+1]
		}
	}
	record := devLogRecord{
		Time:   time.Now().UTC(),
		Source: "tui",
		Level:  "info",
		Event:  event,
		Fields: fields,
	}
	devLogTarget.mutex.Lock()
	defer devLogTarget.mutex.Unlock()
	_ = devLogTarget.encoder.Encode(record)
}

func closeDevLog() {
	if devLogTarget == nil {
		return
	}
	devLogTarget.mutex.Lock()
	defer devLogTarget.mutex.Unlock()
	_ = devLogTarget.closer.Close()
}
