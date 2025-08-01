// Package state defines stared program state.
package state

import (
	"time"

	"go.uber.org/zap"

	"sync2kindle/config"
)

// LocalEnv keeps everything program needs in a single place.
type LocalEnv struct {
	Start         time.Time
	Cfg           *config.Config
	Rpt           *config.Report
	Log           *zap.Logger
	RestoreStdLog func()
}

// NewLocalEnv creates LocalEnv and initializes it.
func NewLocalEnv() *LocalEnv {
	return &LocalEnv{Start: time.Now()}
}

type LocalEnvKeyType string

const (
	EnvValue LocalEnvKeyType = "$-localenv-$"
)

// Set implements cli's flag interface
func (e *LocalEnv) Set(value string) error {
	panic("localenv value should never be set directly")
}

// String implements cli's flag interface
func (e *LocalEnv) String() string {
	return "local-env"
}
