package repo

// Transaction is an abstraction of transactions typically found in database systems.
// One of Commit() or Rollback() must be called on each transaction.
type Transaction interface {
	// Commit will persist the changes in the transaction.
	Commit() error

	// Rollback undoes the changes in a transaction
	Rollback() error
}

type TransactionFactory func() (Transaction, error)
