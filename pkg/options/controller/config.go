package controller

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type QueueStrategy string

const (
	FIFO QueueStrategy = "FIFO"
	FAIR QueueStrategy = "FAIR"
)

type QueueConfig struct {
	Strategy        QueueStrategy `json:"strategy"`
	MaxRunningCount int           `json:"maxRunningCount"`
}

type StepConfig struct {
	// 本来想叫type，但type 是关键字
	Kind        string          `json:"kind"`
	Qps         float64         `json:"qps"`
	SyncTimeout metav1.Duration `json:"syncTimeout"`
	Concurrency int             `json:"concurrency"`
}

type Config struct {
	SyncTimeout metav1.Duration `json:"syncTimeout"`
	Concurrency int             `json:"concurrency"`
	Steps       []StepConfig    `json:"steps"`
	Queue       QueueConfig     `json:"queue"`
}

func NewDefaultConfig() *Config {
	opt := &Config{
		SyncTimeout: metav1.Duration{Duration: 1 * time.Minute},
		Concurrency: 30,
		Steps:       []StepConfig{},
		Queue: QueueConfig{
			Strategy:        FIFO,
			MaxRunningCount: 100,
		},
	}
	return opt
}
