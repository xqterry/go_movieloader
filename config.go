package main

import (
	"log"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

type MovieFileConfig struct {
	Name string		`yaml:"name"`
	Filename string	`yaml:"file"`
	Skip string		`yaml:"ss"`
}

type MoviesConfig struct {
	Movies []MovieFileConfig	`yaml:"movies"`
	Ffmpeg_params string 		`yaml:"params"`
	Crop_size int				`yaml:"crop"`
}

//func NewMovieFileConfig(fn string) *MovieFileConfig {
//	conf := &MovieFileConfig{}
//	log.Println("movie from string ", fn)
//	return conf
//}


func NewMoviesConfig(fn string) *MoviesConfig {
	conf := &MoviesConfig{}
	y, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Println("Load movies config failed")
		return nil
	}
	//log.Printf("%v\n", string(y))
	err = yaml.Unmarshal(y, conf)
	//err = json.Unmarshal(y, conf)
	if err != nil{
		log.Println("parsing movies config file failed", err)
		return nil
	}
	return conf
}

func test() {
	a := MoviesConfig{
		[]MovieFileConfig{
			{"a", "fa", "10"},
			{"b", "fb", "20"},
		},
		"-abc",
		888,
	}
	log.Printf("See test %v", a)
	sa, err := yaml.Marshal(&a)
	log.Println(string(sa), err)

	//ja, err := json.Marshal(&a)
	//log.Println(string(ja), ja, err)

	log.Println("Can I access Config ? ", Config)
}