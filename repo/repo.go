package repo

import "errors"

// Transaction is an abstraction of transactions typically found in database systems.
// One of Commit() or Rollback() must be called on each transaction.
type Transaction interface {
	// Commit will persist the changes in the transaction.
	Commit() error

	// Rollback undoes the changes in a transaction
	Rollback() error
}

type TransactionFactory func() (Transaction, error)

// InMemTransaction satisifies the Transaction interface for in-memory systems.
// However, the only thing it really does is ensure that the same transaction is
// can't be committed/rolled back more than once. As such, this can lead to data
// corruption and should not be used in production systems.
type InMemTransaction bool

func InMemTransactionFactory() (Transaction, error) {
	return new(InMemTransaction), nil
}

func (i *InMemTransaction) Commit() error {
	return i.commitOrRollback()
}

func (i *InMemTransaction) Rollback() error {
	return i.commitOrRollback()
}

func (i *InMemTransaction) commitOrRollback() error {
	if *i {
		return errors.New("Already committed/rolled-back.")
	}
	*i = true
	return nil
}
