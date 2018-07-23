package main

import (
	"runtime"
	"log"
	"sync"
	"time"
)

func main() {
	runtime.GOMAXPROCS(4)

	wg := sync.WaitGroup{}
	wg.Add(1)

	done := make(chan int)
	done2 := make(chan int)

	go func() {
		for {
			select {
			case item := <-done:
				log.Println("done1", item)
			case <-done2:
				log.Println("done2")
			default:
				//break
			}
			time.Sleep(time.Microsecond * 200)
			runtime.Gosched()
		}
	}()

	wg.Wait()
}