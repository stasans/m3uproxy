package streamserver

import (
	"log"
	"sync"
)

func monitorWorker(stream <-chan *streamEntry, stop <-chan bool, wg *sync.WaitGroup) {

	defer wg.Done()
	for s := range stream {
		select {
		case <-stop:
			return
		default:
			s.sources.HealthCheck(Config.Timeout)
			if s.sources.GetActiveSource() == nil {
				continue
			}
			if !s.sources.Active() {
				log.Printf("Stream %s is not active\n", s.sources.GetActiveSource().MediaName())
			}
		}
	}
}
