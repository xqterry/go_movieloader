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
	"os/exec"
	"math/rand"
	"encoding/hex"
//	"bufio"
	"bufio"
	"strings"
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
	worker *zmq.Socket
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

	// WTF!
	// this program will send very large amount ffmpeg data and high memory usage
	// so GC will be triggered and it will purge router/backend.
	// the following unused code just tell GC to keep them.
	router.Close()
	backend.Close()
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

		log.Println("got message", wid, reflect.TypeOf(content), len(content),
			reflect.TypeOf(identity), len(identity),
			"msg size", len(msg))

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
			worker: worker,
		}

		svc.queue <- item
	}
}

func (svc *ZMQService) RandomCrop(src []byte, dst []byte, h int, w int, ch int, cw int, s int) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	bw := w - cw
	bh := h - ch

	aw := r.Intn(bw)
	ah := r.Intn(bh)
	ah += ah % 2

	// keep H % 2 == 0
	if ah == bh {
		ah = bh - 2
	}

	for y := ah; y < ah + ch; y++ {
		for x := aw; x < aw + cw; x++ {
			di := (y - ah) * cw * s + (x - aw) * s
			si := (y * w + x) * s

			dst[di] = src[si]
			dst[di + 1] = src[si + 1]
			dst[di + 2] = src[si + 2]
		}
	}

	//return ah, aw
}

func (svc *ZMQService) _exec_cmd(cmd *exec.Cmd, item *WorkItem, frameWidth int, frameHeight int) (frameCount int) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("get pipe failed \n")
	}

	//stderrIn, err := cmd.StderrPipe()
	//if err != nil {
	//	log.Printf("get err pipe failed \n")
	//}

	defer cmd.Wait()

	outFrameSize := item.cmd.CropW * item.cmd.CropH* 3
	frameSize := frameWidth * frameHeight * 3
	frameBuf := make([]byte, frameSize, frameSize)

	nBytes, nChunks := 0, 0
	_ = nChunks
	r := bufio.NewReader(stdout)
	//r := stdout

	log.Println(": OutFrame size ", outFrameSize, "ffmpeg frame size ", frameSize)
	buf := make([]byte, outFrameSize, outFrameSize)

	_ = cmd.Start()

	//go func() {
	//	stderr, errStderr = copyAndCapture(os.Stderr, stderrIn)
	//}()


	framePos := 0
	for {
		//log.Println("blocking for read")
		//n, err := r.Read(buf[:cap(buf)])
		//log.Println("-> reading # ", item.workerId)

		n, err := r.Read(frameBuf[framePos:frameSize])
		//log.Println("<- reading done # ", item.workerId, n)
		if n == 0 {
			if err == nil {
				log.Println("No err but no data")
				continue
			}
			if err == io.EOF {
				log.Println("fuck eof stdout")
				break
			}
			log.Fatal(err)
		}
		framePos += n
		if framePos < frameSize {
			//log.Println(item.workerId, "Read ", framePos, " need ", frameSize)
			continue
		} else {
			//log.Println("frame", nChunks, " complete")
		}

		nChunks++
		framePos = 0

		if item.cmd.CenterCrop == false {
			// Random Crop
			svc.RandomCrop(frameBuf, buf, frameHeight, frameWidth, item.cmd.CropH, item.cmd.CropW, 3)
		} else {
			copy(buf[:frameSize], frameBuf)
		}

		//log.Println(item.workerId, "got frame", nChunks, " crop size", " len ", len(buf))

		//bigdata[nBytes:n] = buf[:n]
		//log.Println("copy start from ", nBytes, "with", len(buf))
		//copy(bigdata[nBytes:nBytes + int64(n)], buf)

		//log.Println("send message", item.cid, reflect.TypeOf(item.cid), len(buf), reflect.TypeOf(buf))

		needSend := outFrameSize
		nSentTotal := 0
		for nSentTotal < needSend {
			nSent, err := item.worker.SendMessage(item.cid, buf[nSentTotal:])
			//log.Println("Sent piece size ", nSent- len(item.cid), " target ", needSend)
			//log.Println("copy end ", len(buf))
			nSentTotal += nSent - len(item.cid)
			// process buf
			if err != nil && err != io.EOF {
				log.Println(err)
			}
		}

		nBytes += needSend
	}

	log.Println("Send bytes N ", nBytes, " frames ", nChunks)
	stdout.Close()

	return nChunks
}

func (svc *ZMQService) process_cmd(item *WorkItem) {
	log.Println("get item from ", item.workerId, "data:", hex.Dump([]byte(item.cid)), item.cmd.Index, item.cmd )

	var conf_array []*MovieFileConfig
	wild := false
	if strings.Index(item.cmd.Movie, "*") != -1 || strings.Index(item.cmd.Movie, "^") != -1{
		wild = true
	}

	if wild == false {
		fileConf := Config.FindMovieFileConfigByName(item.cmd.Movie)
		if fileConf == nil {
			log.Println("conf not found ", item.cmd.Movie)
			return
		}
		conf_array = append(conf_array, fileConf)
	} else {
		conf_array = Config.FindMatchedConfigs(item.cmd.Movie)
	}

	check_array := make([]*MovieFileConfig, 0)
	for _, conf := range conf_array {
		if _, err := os.Stat(conf.Filename); os.IsNotExist(err) {
			if conf.Type != "images" {
				continue
			}
		}
		check_array = append(check_array, conf)
	}

	conf_array = check_array

	if len(conf_array) == 0 {
		log.Println("find data config count ", len(conf_array))
		return
	}

	for _, c := range conf_array {
		log.Println("Conf file ", c.Filename)
	}

	//scale := fmt.Sprintf("scale=%d:%d", item.cmd.Width, item.cmd.Height)

	// count is 0 return all frames (only for evaluation and short samples)
	pc := item.cmd.Count
	if len(conf_array) > 0 {
		pc = item.cmd.Count / len(conf_array)
	}
	mod := 0
	if pc != 0 {
		mod = item.cmd.Count % pc
	}
	var cc []int
	for _,_ = range(conf_array) {
		cc = append(cc, pc)
	}

	if mod != 0 {
		cc[0] += mod
	}

	actualFrames := 0
	totalFrames := item.cmd.Count
	runOnce := true

	for runOnce || actualFrames < totalFrames {
		runOnce = false
		for i, fileConf := range (conf_array) {
			nFrameCount := cc[i]
			if totalFrames - actualFrames < nFrameCount {
				nFrameCount = totalFrames - actualFrames
			}
			frameCount := fmt.Sprintf("%d", nFrameCount)
			//ss := fileConf.Skip
			ss := fileConf.Skip[rand.Intn(len(fileConf.Skip))]
			fn := fileConf.Filename

			frameWidth := item.cmd.Width
			frameHeight := item.cmd.Height

			if item.cmd.CenterCrop {
				frameWidth = item.cmd.CropW
				frameHeight = item.cmd.CropH
			}

			if frameWidth > fileConf.Width {
				frameWidth = fileConf.Width
			}

			if frameHeight > fileConf.Height {
				frameHeight = fileConf.Height
			}

			vf := fmt.Sprintf("crop=%d:%d", frameWidth, frameHeight)

			need_scale := false
			if item.cmd.CropW > frameWidth || item.cmd.CropH > frameHeight {
				need_scale = true
				frameWidth = item.cmd.CropW
				frameHeight = item.cmd.CropH
			}

			if (item.cmd.Scale && fileConf.Height > 1080) || need_scale {
				vf = fmt.Sprintf("scale=%d:%d", item.cmd.Width, item.cmd.Height)
				frameWidth = item.cmd.Width
				frameHeight = item.cmd.Height
			}
			vf = fmt.Sprintf("%s,select='eq(pict_type\\, I)'", vf)

			args := make([]string, 0)

			if nFrameCount > 0 {
				if fileConf.Type != "images" {
					args = append(args, "-ss")
					args = append(args, ss)
				} else if fileConf.Count > cc[i] {
					args = append(args, "-start_number")
					sn := rand.Intn(fileConf.Count - cc[i])
					ssn := fmt.Sprintf("%d", sn)
					args = append(args, ssn)
				}
			}

			if fileConf.Type == "yuv" {
				args = append(args, "-pixel_format")
				args = append(args, "yuv422p")

				args = append(args, "-video_size")
				args = append(args, fmt.Sprintf("%dx%d", fileConf.Width, fileConf.Height))

				args = append(args, "-framerate")
				args = append(args, fmt.Sprintf("%d", fileConf.FrameRate))
			}

			args = append(args, "-f")
			args = append(args, "rawvideo")



			args = append(args, "-i")
			args = append(args, fn)
			args = append(args, "-f")
			args = append(args, "image2pipe")

			args = append(args, "-vf")
			args = append(args, vf)

			if nFrameCount > 0 {
				args = append(args, "-frames")
				args = append(args, frameCount)
			}

			args = append(args, "-c:v")
			args = append(args, "rawvideo")
			args = append(args, "-pix_fmt")
			args = append(args, "rgb24")
			args = append(args, "pipe:1")

			iargs := make([]interface{}, len(args))
			for i, v := range args {
				iargs[i] = v
			}
			log.Println(iargs...)

			cmd := exec.Command("ffmpeg", args...)
			//cmd.Stdout = os.Stdout
			//cmd.Stderr = os.Stderr
			//cmd.Run()

			nFrames := svc._exec_cmd(cmd, item, frameWidth, frameHeight)
			log.Println(fileConf.Name, nFrames)

			actualFrames += nFrames

			if actualFrames >= totalFrames {
				break
			}
		}
	}

	if item.cmd.Count == 0 {
		nSent, _ := item.worker.SendMessage(item.cid, []byte("FEND"))
		log.Println("sent all ", nSent, actualFrames)
	}
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
			go svc.process_cmd(item)
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
		conn.SetWindowSize(0, 0)
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
