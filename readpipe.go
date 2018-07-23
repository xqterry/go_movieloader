package main

import (
	"io"
	"log"
	"os/exec"
	"os"
)

func copyAndCapture(w io.Writer, r io.Reader) ([]byte, error) {
	return nil, nil
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			_, err := w.Write(d)
			if err != nil {
				return out, err
			}
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return out, err
		}
	}
	// never reached
	panic(true)
	return nil, nil
}

func main() {

	var stderr []byte
	var errStderr error

	var bigdata [896 * 896 * 3 * 10000]byte

	//cmd := exec.Command("ls", "-l")
	cmd := exec.Command("ffmpeg", /* "-ss",  "00:01:00", */ "-i", "/dataset/INSURGENT.Left_Right.mkv",
		"-vf", "crop=896:512", "-f", "image2pipe", "-frames", "10", "-c:v", "rawvideo", "-pix_fmt", "rgb24",  "-")
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	//cmd.Run()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("get pipe failed \n")
	}

	stderrIn, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("get err pipe failed \n")
	}

	defer cmd.Wait()

	nBytes, nChunks := int64(0), int64(0)
	//r := bufio.NewReader(stdout)
	r := stdout
	buf := make([]byte, 0, 4*1024)

	_ = cmd.Start()

	go func() {
		stderr, errStderr = copyAndCapture(os.Stderr, stderrIn)
	}()


	for {
		log.Println("blocking for read")
		n, err := r.Read(buf[:cap(buf)])
		log.Println("reading done")
		buf = buf[:n]
		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		nChunks++
		//bigdata[nBytes:n] = buf[:n]
		log.Println("copy start from ", nBytes, "with", len(buf))
		copy(bigdata[nBytes:nBytes + int64(n)], buf)
		log.Println("copy end ", len(buf))
		nBytes += int64(len(buf))
		// process buf
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

	}
	frames := nBytes / 3 / 896 / 512
	log.Println("Bytes:", nBytes, "Chunks:", nChunks, "Frames count:", frames)

	errStr := string(stderr)
	log.Println("err", errStr)
}
