package gocb

import (
	"math"
	"time"
)

// QueryRetryBehavior defines the behavior to be used when retrying queries
type QueryRetryBehavior interface {
	NextInterval(retries uint) time.Duration
	CanRetry(retries uint) bool
}

// QueryRetryDelayFunction is called to get the next try delay
type QueryRetryDelayFunction func(retryDelay uint, retries uint) time.Duration

// QueryLinearDelayFunction provides retry delay durations (ms) following a linear increment pattern
func QueryLinearDelayFunction(retryDelay uint, retries uint) time.Duration {
	return time.Duration(retryDelay*retries) * time.Millisecond
}

// QueryExponentialDelayFunction provides retry delay durations (ms) following an exponential increment pattern
func QueryExponentialDelayFunction(retryDelay uint, retries uint) time.Duration {
	pow := math.Pow(float64(retryDelay), float64(retries))
	return time.Duration(pow) * time.Millisecond
}

// QueryDelayRetryBehavior provides the behavior to use when retrying queries with a backoff delay
type QueryDelayRetryBehavior struct {
	maxRetries uint
	retryDelay uint
	delayLimit time.Duration
	delayFunc  QueryRetryDelayFunction
}

// NewQueryDelayRetryBehavior provides a QueryDelayRetryBehavior that will retry at most maxRetries number of times and
// with an initial retry delay of retryDelay (ms) up to a maximum delay of delayLimit
func NewQueryDelayRetryBehavior(maxRetries uint, retryDelay uint, delayLimit time.Duration, delayFunc QueryRetryDelayFunction) *QueryDelayRetryBehavior {
	return &QueryDelayRetryBehavior{
		retryDelay: retryDelay,
		maxRetries: maxRetries,
		delayLimit: delayLimit,
		delayFunc:  delayFunc,
	}
}

// NextInterval calculates what the next retry interval (ms) should be given how many
// retries there have been already
func (rb *QueryDelayRetryBehavior) NextInterval(retries uint) time.Duration {
	interval := rb.delayFunc(rb.retryDelay, retries)
	if interval > rb.delayLimit {
		interval = rb.delayLimit
	}

	return interval
}

// CanRetry determines whether or not the query can be retried according to the behavior
func (rb *QueryDelayRetryBehavior) CanRetry(retries uint) bool {
	return retries < rb.maxRetries
}
