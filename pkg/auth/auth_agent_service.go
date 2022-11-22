package auth

//go:generate moq -out auth_agent_service_moq.go . AuthAgentService
type AuthAgentService interface {
	GetClientID(clusterID string) (string, error)
}
