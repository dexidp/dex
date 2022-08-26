package mongo

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Session struct {
	session mongo.Session
	txnOpts *options.TransactionOptions
}

func NewSession(mongoSession mongo.Session, transactionOptions *options.TransactionOptions) *Session {
	return &Session{
		session: mongoSession,
		txnOpts: transactionOptions,
	}
}

func (s *Session) End(c context.Context) {
	s.session.EndSession(c)
}

func (s *Session) Operate(c context.Context, operation func(ctx context.Context) error) error {
	if err := s.session.StartTransaction(s.txnOpts); err != nil {
		return errors.Wrap(err, "unable to start a mongo transaction")
	}

	if err := mongo.WithSession(c, s.session, func(sc mongo.SessionContext) error {
		return operation(sc)
	}); err != nil {
		if errT := s.session.AbortTransaction(c); errT != nil {
			return errors.Wrap(err, errT.Error())
		}

		return err
	}

	if err := s.session.CommitTransaction(c); err != nil {
		return errors.Wrap(err, "unable to commit a mongo transaction")
	}

	return nil
}
