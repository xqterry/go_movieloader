package main

import (
	zmq "github.com/pebbe/zmq4"
	"os"
	"io"
	"fmt"
	"log"
	"gopkg.in/yaml.v2"
	"time"
	"sync"
	"reflect"
	"bytes"
	"encoding/gob"
	"gopkg.in/xtaci/kcp-go.v3"
	"github.com/davecgh/go-spew/spew"
	"runtime"
)

const (
	CHUNK_SIZE = 250000
)

func client_thread(pipe chan<- string) {
	dealer, _ := zmq.NewSocket(zmq.DEALER)
	dealer.Connect("tcp://127.0.0.1:6000")

	dealer.Send("fetch", 0)
	total := 0  //  Total bytes received
	chunks := 0 //  Total chunks received

	for {
		frame, err := dealer.RecvBytes(0)
		if err != nil {
			break //  Shutting down, quit
		}
		chunks++
		size := len(frame)
		total += size
		if size == 0 {
			break //  Whole file received
		}
	}
	fmt.Printf("%v chunks received, %v bytes\n", chunks, total)
	pipe <- "OK"
}

type DataClient struct {
	cid string
}

func server_thread() {
	file, err := os.Open("testdata")
	if err != nil {
		panic(err)
	}

	router, _ := zmq.NewSocket(zmq.ROUTER)
	//  Default HWM is 1000, which will drop messages here
	//  since we send more than 1,000 chunks of test data,
	//  so set an infinite HWM as a simple, stupid solution:
	router.SetRcvhwm(0)
	router.SetSndhwm(0)
	router.Bind("tcp://*:6000")
	for {
		//  First frame in each message is the sender identity
		identity, err := router.Recv(0)
		if err != nil {
			fmt.Println("msg recv error")
			break //  Shutting down, quit
		}

		fmt.Printf("recv from %s\n", identity)

		//  Second frame is "fetch" command
		command, _ := router.Recv(0)
		if command != "fetch" {
			panic("command != \"fetch\"")
		}

		chunk := make([]byte, CHUNK_SIZE)
		for {
			n, _ := io.ReadFull(file, chunk)
			router.SendMessage(identity, chunk[:n])
			if n == 0 {
				break //  Always end with a zero-size frame
			}
		}
	}
	file.Close()
}

//func main() {
//	pipe := make(chan string)
//
//	//  Start child threads
//	go server_thread()
//	go client_thread(pipe)
//	//  Loop until client tells us it's done
//	<-pipe
//
//	//cmd := exec.Command("ffmpeg", "-i", "/dataset/INSURGENT.Left_Right.mkv", "-f image2pipe", "-")
//	//stdout, err := cmd.StdoutPipe()
//	//if err != nil {
//	//	log.Fatal(err)
//	//}
//	//if err := cmd.Start(); err != nil {
//	//	log.Fatal(err)
//	//}
//}

type ZMQService struct {
	sid int
	port uint16
	router *zmq.Socket
	backend *zmq.Socket
	queue chan *WorkItem
	queueWg *sync.WaitGroup
}

type WorkItem struct {
	cid string
	cmd *SendCommand

	workerId int
}


func NewZMQService(port uint16) *ZMQService {
	s := ZMQService{}
	s.port = port
	s.queue = make(chan *WorkItem, 100)
	s.queueWg = &sync.WaitGroup{}
	return &s
}

func (svc *ZMQService) InitQueues(n int)  {
	for i := 0; i < n; i ++ {
		svc.queueWg.Add(1)
		go svc.process_queue(i + 1)
	}

	svc.queueWg.Wait()

	fmt.Println("process queues are all ready")

}

func (svc *ZMQService) Start() {
	fmt.Println("Start service @ :", svc.port)

	zmq.SetIoThreads(10)

	router, err := zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		fmt.Println("create router socket failed", err)
	}

	router.SetRcvhwm(0)
	router.SetSndhwm(0)
	endpoint := fmt.Sprintf("tcp://*:%d", svc.port)
	err = router.Bind(endpoint)

	if err != nil {
		fmt.Println("binding router socket failed", err)
	}

	svc.router = router

	//  Backend socket talks to workers over inproc
	backend, err := zmq.NewSocket(zmq.DEALER)
	//backend, err := zmq.NewSocket(zmq.POLLIN)
	if err != nil {
		fmt.Println("create IPC socket failed", err)
	}
	//defer backend.Close()
	backend.Bind("inproc://oct_backend")
	if err != nil {
		fmt.Println("bind IPC socket failed", err)
	}

	//  Launch pool of worker threads, precise number is not critical
	for i := 0; i < 50; i++ {
		svc.queueWg.Add(1)
		go svc.server_worker(i)
	}

	svc.queueWg.Wait()

	log.Println("try to start proxy")

	//  Connect backend to frontend via a proxy
	err = zmq.Proxy(router, backend, nil)
	log.Fatalln("Proxy interrupted:", err)

}

func (svc *ZMQService) server_worker(wid int) {
	//context, _ := zmq.NewContext()

	worker, err := zmq.NewSocket(zmq.DEALER)
	//worker, err := context.NewSocket(zmq.DEALER)
	if err != nil {
		log.Fatalln("create worker socket failed")
	}
	defer worker.Close()
	err = worker.Connect("inproc://oct_backend")

	if err != nil {
		log.Fatalln("connect ipc worker failed", err)
	}

	svc.queueWg.Done()

	for {
		//  The DEALER socket gives us the reply envelope and message
		log.Println("recv start")
		msg, _ := worker.RecvMessage(0)
		identity, content := pop(msg)

		log.Println("got message", wid, reflect.TypeOf(content), len(content))
		////  Send 0..4 replies back
		//replies := 1 //rand.Intn(5)
		//for reply := 0; reply < replies; reply++ {
		//	//  Sleep for some fraction of a second
		//	time.Sleep(time.Duration(rand.Intn(1000)+1) * time.Millisecond)
		//	worker.SendMessage(identity, content)
		//}
		cmd := &SendCommand{}

		buf := &bytes.Buffer{}
		gob.NewEncoder(buf).Encode(content[0])
		bs := buf.Bytes()

		bs = []byte(content[0])
		err := yaml.Unmarshal(bs, cmd)
		if err != nil {
			log.Println("recv cmd error", err)
		}

		item := &WorkItem{
			cid: identity[0],
			cmd: cmd,
		}

		svc.queue <- item
	}
}

func (svc *ZMQService) process_cmd(item *WorkItem) {
	log.Println("get item from ", item.workerId, "data:", item.cid, item.cmd.Index )

}


func (svc *ZMQService) process_queue(pid int) {
	var item *WorkItem

	log.Println("process queue id ", pid, "start")
	svc.queueWg.Done()

	for {
		item = nil
		select {
		case item = <- svc.queue:
			//break
		//default:
			//break
		}

		if item != nil {
			item.workerId = pid
			svc.process_cmd(item)
		} else {
			time.Sleep(time.Microsecond * 10)
			runtime.Gosched()
			//println("no item ", pid)
		}

		time.Sleep(time.Microsecond * 10)
		runtime.Gosched()
	}
}

func (svc *ZMQService) KCPStart() {
	laddr := fmt.Sprintf(":%d", svc.port)
	lis, err := kcp.ListenWithOptions(laddr, nil, 10, 3)

	if err != nil {
		spew.Dump(err)
	}

	log.Println("Server started")

	for {
		conn, err := lis.AcceptKCP()

		if err != nil {
			spew.Dump("AcceptKCP failed", err)
		}

		log.Println("remote address:", conn.RemoteAddr())
		conn.SetStreamMode(true)
		conn.SetWriteDelay(true)
		// conn.SetNoDelay(config.NoDelay, config.Interval, config.Resend, config.NoCongestion)
		// conn.SetMtu(config.MTU)
		conn.SetWindowSize(1000, 1000)
		// conn.SetACKNoDelay(config.AckNodelay)

		go func(conn *kcp.UDPSession) {
			for {
				b := make([]byte, 11)
				num, err := conn.Read(b)

				if err != nil {
					spew.Dump(num, err)
				}

				if num != 0 {
					spew.Dump(b)

					cmd := &SendCommand{}

					item := &WorkItem{
						cid: "abc",
						cmd: cmd,
					}

					svc.queue <- item
				}

				log.Println("reading empty?")
			}
		}(conn)

		time.Sleep(time.Microsecond * 10)
	}

}

func pop(msg []string) (head, tail []string) {
	if msg[1] == "" {
		head = msg[:2]
		tail = msg[2:]
	} else {
		head = msg[:1]
		tail = msg[1:]
	}
	return
}