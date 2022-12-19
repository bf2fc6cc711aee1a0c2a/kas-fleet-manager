package quota_management

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

func Test_UnmarshalJson(t *testing.T) {

	cetTimezone, _ := time.LoadLocation("CET")
	gmtTimezone, _ := time.LoadLocation("GMT")

	tests := []struct {
		name    string
		value   string
		expect  time.Time
		wantErr bool
	}{
		{
			name:    "Test valid date 01:00 (CET)",
			value:   `"2022-12-06 +01:00"`,
			expect:  time.Date(2022, 12, 06, 0, 0, 0, 0, cetTimezone),
			wantErr: false,
		},
		{
			name:    "Test valid date +00:00",
			value:   `"2022-12-06 +00:00"`,
			expect:  time.Date(2022, 12, 06, 0, 0, 0, 0, gmtTimezone),
			wantErr: false,
		},
		{
			name:    "Test invalid date (no TZ)",
			value:   `"2022-12-06"`,
			wantErr: true,
		},
		{
			name:    "Test invalid date (invalid TZ)",
			value:   `"2022-12-06 INVALID"`,
			wantErr: true,
		},
		{
			name:    "Test invalid date (invalid month)",
			value:   `"2022-14-06 CET"`,
			wantErr: true,
		},
		{
			name:    "Test invalid date (invalid separator)",
			value:   `"2022/14/06 CET"`,
			wantErr: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			var dt ExpirationDate
			err := json.Unmarshal([]byte(tt.value), &dt)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr), "Error: %v", err)
			g.Expect(time.Time(dt)).To(gomega.BeTemporally("==", tt.expect))
		})
	}
}

func Test_MarshalJson(t *testing.T) {

	cetTimezone, _ := time.LoadLocation("CET")
	gmtTimezone, _ := time.LoadLocation("GMT")

	tests := []struct {
		name    string
		value   ExpirationDate
		expect  string
		wantErr bool
	}{
		{
			name:    "Test valid date tz:+01:00 (CET)",
			value:   ExpirationDate(time.Date(2022, 12, 06, 0, 0, 0, 0, cetTimezone)),
			expect:  `"2022-12-06 +01:00"`,
			wantErr: false,
		},
		{
			name:    "Test valid date tz:+00:00 (GMT)",
			value:   ExpirationDate(time.Date(2022, 12, 06, 0, 0, 0, 0, gmtTimezone)),
			expect:  `"2022-12-06 +00:00"`,
			wantErr: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			val, err := json.Marshal(&tt.value)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))

			stringVal := string(val)
			g.Expect(stringVal).To(gomega.Equal(tt.expect))
		})
	}
}

func Test_MarshalYAML(t *testing.T) {

	cetTimezone, _ := time.LoadLocation("CET")
	gmtTimezone, _ := time.LoadLocation("GMT")

	tests := []struct {
		name    string
		value   ExpirationDate
		expect  string
		wantErr bool
	}{
		{
			name:    "Test valid date tz: +01:00 (CET)",
			value:   ExpirationDate(time.Date(2022, 12, 06, 0, 0, 0, 0, cetTimezone)),
			expect:  "2022-12-06 +01:00",
			wantErr: false,
		},
		{
			name:    "Test valid date tz: +00:00 GMT",
			value:   ExpirationDate(time.Date(2022, 12, 06, 0, 0, 0, 0, gmtTimezone)),
			expect:  "2022-12-06 +00:00",
			wantErr: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			val, err := yaml.Marshal(&tt.value)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))

			stringVal := string(val)
			g.Expect(strings.TrimSuffix(stringVal, "\n")).To(gomega.Equal(tt.expect))
		})
	}
}

func Test_UnmarshalYAML(t *testing.T) {
	cetTimezone, _ := time.LoadLocation("CET")
	gmtTimezone, _ := time.LoadLocation("GMT")

	tests := []struct {
		name    string
		value   string
		expect  time.Time
		wantErr bool
	}{
		{
			name:    "Test valid date tz:+01:00 (CET) - no quotes",
			value:   "2022-12-06 +01:00",
			expect:  time.Date(2022, 12, 06, 0, 0, 0, 0, cetTimezone),
			wantErr: false,
		},
		{
			name:    "Test valid date tz:+01:00 (CET) - with quotes",
			value:   `"2022-12-06 +01:00"`,
			expect:  time.Date(2022, 12, 06, 0, 0, 0, 0, cetTimezone),
			wantErr: false,
		},
		{
			name:    "Test valid date tz:+00:00 GMT",
			value:   `"2022-12-06 +00:00"`,
			expect:  time.Date(2022, 12, 06, 0, 0, 0, 0, gmtTimezone),
			wantErr: false,
		},
		{
			name:    "Test invalid date (no TZ)",
			value:   `"2022-12-06"`,
			wantErr: true,
		},
		{
			name:    "Test invalid date (invalid TZ)",
			value:   `"2022-12-06 INVALID"`,
			wantErr: true,
		},
		{
			name:    "Test invalid date (invalid month)",
			value:   `"2022-14-06 CET"`,
			wantErr: true,
		},
		{
			name:    "Test invalid date (invalid separator)",
			value:   `"2022/14/06 CET"`,
			wantErr: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			var dt ExpirationDate
			err := yaml.Unmarshal([]byte(tt.value), &dt)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(time.Time(dt)).To(gomega.BeTemporally("==", tt.expect))
		})
	}
}
