package redis

import (
	"log"
	"time"
)

type ClientClosedError struct{}

func (m *ClientClosedError) Error() string {
	return "redis connection limiter has been closed"
}

type limiter struct {
	backoffDuration     time.Duration
	connectionFailures  int
	connectionStateChan chan ConnectionState
	closed              bool
}

func newLimiter(backoffDuration time.Duration, connectionStateChan chan ConnectionState) *limiter {
	connectionFailures := -1
	return &limiter{
		backoffDuration:     backoffDuration,
		connectionFailures:  connectionFailures,
		connectionStateChan: connectionStateChan,
		closed:              false,
	}
}

func (l *limiter) close() {
	l.closed = true
}

func (l *limiter) wait() {
	// FIXME: because Allow can be called concurrently, we should linearize this though a mutex so one routine can wait
	//  for a stopped signal and terminate immediately instead of waiting for the next Allow call and l.closed check
	time.Sleep(l.backoffDuration)
}

func (l *limiter) Allow() error {
	if l.closed {
		return &ClientClosedError{}
	}

	if l.connectionFailures > 0 {
		log.Printf("last connection attempt failed, backing off for %v\n", l.backoffDuration)
		l.wait()
	}

	return nil
}

func (l *limiter) ReportResult(err error) {
	if err == nil {
		if l.connectionFailures == -1 {
			// first connection attempt succeeded
			log.Println("connected to redis")
			l.connectionStateChan <- Connected
		}
		if l.connectionFailures > 0 {
			// report only on first connection (-1) and on recovery (> 0)
			log.Println("redis connection recovered from failure state")
			l.connectionStateChan <- Recovered
		}
		l.connectionFailures = 0
	} else {
		l.connectionFailures++
		if l.connectionFailures == 1 {
			// only report first connection error
			log.Println("redis connection is in failure state")
			l.connectionStateChan <- Failed
		}
	}
}
