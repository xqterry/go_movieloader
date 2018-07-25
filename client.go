package main

import (
	zmq "github.com/pebbe/zmq4"
	"sync"
	"time"
	"fmt"
	"math/rand"
	"log"
	"runtime"
	"gopkg.in/yaml.v2"
	"reflect"
)

func set_id(soc *zmq.Socket) string {
	identity := fmt.Sprintf("%04X-%04X", rand.Intn(0x10000), rand.Intn(0x10000))
	soc.SetIdentity(identity)

	return identity
}

func client_task() {
	var mu sync.Mutex

	client, _ := zmq.NewSocket(zmq.DEALER)
	defer client.Close()

	//  Set random identity to make tracing easier
	my_id := set_id(client)
	state := 0
	client.Connect("tcp://localhost:6000")

	go func() {
		for request_nbr := 1; true; request_nbr++ {
			time.Sleep(time.Second * 3)
			if state != 0 {
				log.Println("state not ready")
				continue
			}
			cmd := SendCommand{
				Code: request_nbr,
				Names: []string{"aa", "bb"},
				Index: request_nbr % 100,
				Count: 10,
				Width: 448,
				Height: 448,
			}
			out, err := yaml.Marshal(&cmd)
			if err != nil {
				log.Println("marshal error ", err)
				continue
			}

			mu.Lock()
			state = 1
			//client.SendMessage(fmt.Sprintf("request #%d", request_nbr))
			client.SendMessage(out)
			log.Println("send ok")
			mu.Unlock()
		}
	}()

	sz := 0
	for {
		//time.Sleep(10 * time.Millisecond)
		mu.Lock()
		msg, err := client.RecvMessage(zmq.DONTWAIT)
		//msg, err := client.RecvMessage(0)
		if err == nil {
			id, _ := client.GetIdentity()
			fmt.Println(len(msg), id, "My ID", my_id)

			sz += len(msg[0])
			log.Println("recv message", reflect.TypeOf(msg[0]), len(msg[0]))
			state = 0

			//break
		}
		mu.Unlock()

		//log.Println("recv size ", sz)
	}
}

func main() {
	runtime.GOMAXPROCS(4)

	ch := make(chan int)

	go client_task()
	go client_task()
	//go client_task()

	log.Println("waiting for clients")

	_ = <- ch
}