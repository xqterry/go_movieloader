package main

import (
	"fmt"
	"flag"
	"runtime"
)

type Args struct {
	configFile string
}

var Config *MoviesConfig = nil

func main() {
	runtime.GOMAXPROCS(4)

	fmt.Println("flying pig starts to fly")

	// parse command line args
	args := Args{}
	flag.StringVar(&args.configFile, "conf", "./config/config.yaml", "config file (.yaml)")

	fmt.Println("Params:", args)

	Config = NewMoviesConfig(args.configFile)
	fmt.Printf("conf: %v\n", Config)
	fmt.Println("crop", Config.Crop_size)

	//test()

	svc := NewZMQService(6000)
	svc.InitQueues(10)
	svc.Start()

	//svc.KCPStart()

}
