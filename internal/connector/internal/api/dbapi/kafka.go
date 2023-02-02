package dbapi

type KafkaConnectionSettings struct {
	KafkaID         string `gorm:"column:id"`
	BootstrapServer string
}
