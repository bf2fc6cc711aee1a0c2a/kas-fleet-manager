package observatorium

import (
	"time"
)

type Configuration struct {
	BaseURL   string
	AuthToken string
	Timeout   time.Duration
	Insecure  bool
	AuthType  string
}
