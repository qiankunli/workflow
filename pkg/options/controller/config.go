package controller

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	SyncTimeout metav1.Duration `json:"syncTimeout"`
	Concurrency int             `json:"concurrency"`
}

func NewDefaultConfig() *Config {
	opt := &Config{
		SyncTimeout: metav1.Duration{Duration: 1 * time.Minute},
		Concurrency: 30,
	}
	return opt
}
