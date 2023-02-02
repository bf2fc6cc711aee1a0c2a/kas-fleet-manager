package dbapi

type ServiceAccount struct {
	ClientId        string
	ClientSecret    string `gorm:"-"`
	ClientSecretRef string `gorm:"column:client_secret"`
}
