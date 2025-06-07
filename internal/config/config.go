package config

import "time"

type (
	Exchanges map[string]string

	Queues map[string]string

	HealthCheck struct {
		Port string
		Use  bool
	}

	RetrierOpts struct {
		MaxRetries int
		Interval   time.Duration
	}

	Urls map[string]string

	Config struct {
		Exchanges   Exchanges
		Queues      Queues
		RetrierOpts RetrierOpts
		Urls        Urls
		HealthCheck HealthCheck
	}
)

func NewConfig() *Config {
	return &Config{
		Exchanges: Exchanges{
			"form":   "form",
			"answer": "answer",
		},
		Queues: Queues{
			"form":   "form",
			"answer": "answer",
		},
		RetrierOpts: RetrierOpts{
			MaxRetries: 3,
			Interval:   5 * time.Second,
		},
		Urls: Urls{
			"rabbitmq": "amqp://rabbitmq:5672",
			"redis":    "redis:6379",
		},
		HealthCheck: HealthCheck{
			Port: "8080",
			Use:  true,
		},
	}
}
