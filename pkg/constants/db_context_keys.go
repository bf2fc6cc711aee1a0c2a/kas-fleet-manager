package constants

type contextKey int
type contextTransactionIDKey string

const (
	TransactionKey   contextKey              = iota
	TransactionIDkey contextTransactionIDKey = "txid"
)
