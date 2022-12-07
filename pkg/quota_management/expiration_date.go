package quota_management

import (
	"encoding/json"
	"gopkg.in/yaml.v2"
	"strings"
	"time"
)

const layout = "2006-01-02 -07:00"

var _ json.Marshaler = &ExpirationDate{}
var _ json.Unmarshaler = &ExpirationDate{}

var _ yaml.Marshaler = &ExpirationDate{}

var _ yaml.Unmarshaler = &ExpirationDate{}

type ExpirationDate time.Time

func (e *ExpirationDate) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`) //get rid of "
	if value == "" || value == "null" {
		return nil
	}

	t, err := time.Parse(layout, value) //parse time
	if err != nil {
		return err
	}
	*e = ExpirationDate(t) //set result using the pointer
	return nil
}

func (e *ExpirationDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(*e).Format(layout) + `"`), nil
}

func (e *ExpirationDate) MarshalYAML() (any, error) {
	return time.Time(*e).Format(layout), nil
}

func (e *ExpirationDate) UnmarshalYAML(unmarshal func(any) error) error {
	var buf string
	err := unmarshal(&buf)
	if err != nil {
		return err
	}

	t, err := time.Parse(layout, buf) //parse time
	if err != nil {
		return err
	}
	*e = ExpirationDate(t) //set result using the pointer
	return nil
}

func (e *ExpirationDate) HasExpired() bool {
	if e == nil {
		// a nil expiration date means 'never expire'
		return false
	}
	return time.Now().After(time.Time(*e))
}
