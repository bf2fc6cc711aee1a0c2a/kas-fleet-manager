package api

type ServiceAccount struct {
	ID           string `json:"id,omitempty"`
	ClientID     string `json:"clientID,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	Name         string `json:"name,omitempty"`
	Owner        string `json:"owner,omitempty"`
	Description  string `json:"description,omitempty"`
}
