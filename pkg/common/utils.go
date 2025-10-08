/*
Copyright © 2024 netr0m <netr0m@pm.me>
*/

package common

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

func InitLogger(debugLogging bool) {
	logrus.AddHook(&Hook{})
	if debugLogging {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) Error() string {
	return fmt.Sprintf("%s failed with status %s: %s", e.Operation, e.Status, e.Message)
}

func (e *Error) Debug() string {
	var debugLines []string

	if e.Request != nil {
		debugLines = append(debugLines, fmt.Sprintf("Request:\n%v", e.Request))
	}
	if e.Response != nil {
		debugLines = append(debugLines, fmt.Sprintf("Response:\n%v", e.Response))
	}
	if e.Err != nil {
		debugLines = append(debugLines, fmt.Sprintf("Error:\n%v", e.Err.Error()))
	}

	return strings.Join(debugLines, "\n")
}

type Hook struct{}

func (h *Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *Hook) Fire(e *logrus.Entry) error {
	for k, v := range e.Data {
		if s, ok := v.(string); ok {
			if s == "" {
				delete(e.Data, k)
				continue
			}
		}
	}
	return nil
}
