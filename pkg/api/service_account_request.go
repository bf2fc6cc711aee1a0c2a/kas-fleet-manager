package api

type ServiceAccountRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
