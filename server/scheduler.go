package server

import (
	"time"
	//	"fmt"
)

type Scheduler struct {
	isRunning bool
	sleepMs time.Duration
}

func schedulerLoop(scheduler *Scheduler) {
	for {
		if (scheduler.isRunning == false) {
			break
		}
		time.Sleep(scheduler.sleepMs * time.Millisecond)
	}
}

func StartScheduler() {
	scheduler := Scheduler{isRunning: true, sleepMs: 1000}

	go schedulerLoop(&scheduler)
}
