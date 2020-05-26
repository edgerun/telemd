package redis

import (
	"log"
	"time"
)

type limiter struct {
	backoffDuration     time.Duration
	connectionFailures  int
	connectionStateChan chan ConnectionState
}

func newLimiter(backoffDuration time.Duration, connectionStateChan chan ConnectionState) *limiter {
	connectionFailures := -1
	return &limiter{
		backoffDuration:     backoffDuration,
		connectionFailures:  connectionFailures,
		connectionStateChan: connectionStateChan,
	}
}

func (l *limiter) Allow() error {
	if l.connectionFailures > 0 {
		log.Printf("last connection attempt failed, backing off for %v\n", l.backoffDuration)
		time.Sleep(l.backoffDuration)
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
		if l.connectionFailures == -1 {
			// don't enter failed state if we had no connection before
			return
		}
		l.connectionFailures++
		if l.connectionFailures == 1 {
			// only report first connection error
			log.Println("redis connection is in failure state")
			l.connectionStateChan <- Failed
		}
	}
}
