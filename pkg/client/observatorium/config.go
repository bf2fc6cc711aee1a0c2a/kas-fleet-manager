package observatorium

import (
	"time"
)

type Configuration struct {
	BaseURL  string
	Timeout  time.Duration
	Insecure bool
}
