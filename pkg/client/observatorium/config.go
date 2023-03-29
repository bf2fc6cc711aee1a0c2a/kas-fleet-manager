package observatorium

import (
	"time"
)

type Configuration struct {
	BaseURL   string
	AuthToken string
	Timeout   time.Duration
	Debug     bool
	Insecure  bool
	AuthType  string
}
