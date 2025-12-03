package repositories

import "log"

type TransactionRepository struct {
    dsn string
}

func NewTransactionRepository(dsn string) *TransactionRepository {
    log.Printf("TransactionRepository using DSN: %s", dsn)
    return &TransactionRepository{dsn: dsn}
}

// TODO: implement persistence for transactions, charges, refunds.
