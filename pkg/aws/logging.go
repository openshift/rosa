/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aws

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
)

type LoggerWrapper struct {
	loggerType   string
	logrusLogger *logrus.Logger
	logrLogger   *logr.Logger
}

func NewLoggerWrapper(logrusLog *logrus.Logger, logrLog *logr.Logger) *LoggerWrapper {
	if logrusLog != nil {
		return &LoggerWrapper{
			loggerType:   "logrus",
			logrusLogger: logrusLog,
		}
	}

	if logrLog != nil {
		return &LoggerWrapper{
			loggerType: "logr",
			logrLogger: logrLog,
		}
	}

	return nil
}

func (lw *LoggerWrapper) GetLevel() (lvl int) {
	switch lw.loggerType {
	case "logrus":
		lvl = int(lw.logrusLogger.GetLevel())
	case "logr":
		lvl = lw.logrLogger.GetV()
	}

	return lvl
}

func (lw *LoggerWrapper) Debug(args ...interface{}) {
	switch lw.loggerType {
	case "logrus":
		lw.logrusLogger.Debug(args...)
	case "logr":
		lw.logrLogger.Info(args[0].(string))
	}
}

func (lw *LoggerWrapper) Info(args ...interface{}) {
	switch lw.loggerType {
	case "logrus":
		lw.logrusLogger.Info(args...)
	case "logr":
		lw.logrLogger.Info(args[0].(string))
	}
}

func (lw *LoggerWrapper) Warn(args ...interface{}) {
	switch lw.loggerType {
	case "logrus":
		lw.logrusLogger.Warn(args...)
	case "logr":
		lw.logrLogger.Info(args[0].(string))
	}
}

func (lw *LoggerWrapper) Error(args ...interface{}) {
	switch lw.loggerType {
	case "logrus":
		lw.logrusLogger.Error(args...)
	case "logr":
		lw.logrLogger.Error(fmt.Errorf("awsClient error"), args[0].(string))
	}
}

func (lw *LoggerWrapper) Fatal(args ...interface{}) {
	switch lw.loggerType {
	case "logrus":
		lw.logrusLogger.Fatal(args...)
	case "logr":
		lw.logrLogger.Error(fmt.Errorf("awsClient error"), args[0].(string))
	}
}
