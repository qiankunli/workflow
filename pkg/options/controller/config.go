package controller

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
}

func NewDefaultConfig() *Config {
	opt := &Config{
		SyncTimeout: metav1.Duration{Duration: 1 * time.Minute},
		Concurrency: 30,
		Steps:       []StepConfig{},
	}
	return opt
}
