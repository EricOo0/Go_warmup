package main

import (
	"design_mode/observer_mode"
	"time"
)

func main() {
	var subject observer_mode.RealSubject
	var objectA observer_mode.ObjectA
	var objectB observer_mode.ObjectB
	objectA.Sub = &subject
	objectB.Sub = &subject
	objectA.Register()
	objectB.Register()
	for {
		time.Sleep(5 * time.Second)
		subject.NotifyObjects()
	}

}
