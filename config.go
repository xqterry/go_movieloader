package main

import (
	"log"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"regexp"
	"math/rand"
	"time"
)

type MovieFileConfig struct {
	Name string		`yaml:"name"`
	Filename string	`yaml:"file"`
	Skip []string		`yaml:"ss"`
	Width int   `yaml:"w"`
	Height int  `yaml:"h"`
	Type string   `yaml:"type"`
	Count int 	`yaml:"count"`
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

func (conf *MoviesConfig) FindMovieFileConfigByName(name string) *MovieFileConfig {
    for _, c := range conf.Movies {
        if c.Name == name {
            return &c
        }
    }
    return nil
}

func (conf *MoviesConfig) FindMatchedConfigs(name string) []*MovieFileConfig {
	var ret []*MovieFileConfig
	for i, c := range conf.Movies {
		m, err := regexp.MatchString(name, c.Name)
		if err != nil {
			continue
		}

		if m {
			ret = append(ret, &conf.Movies[i])
		}
	}

	rand.Seed(time.Now().Unix())
	for i := len(ret) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		ret[i], ret[j] = ret[j], ret[i]
	}

	return ret
}

//func test() {
//	a := MoviesConfig{
//		[]MovieFileConfig{
//			{"a", "fa", []string {"10"}, 800, 600, "video"},
//			{"b", "fb", []string {"20"}, 800, 600, "image"},
//		},
//		"-abc",
//		888,
//	}
//	log.Printf("See test %v", a)
//	sa, err := yaml.Marshal(&a)
//	log.Println(string(sa), err)
//
//	//ja, err := json.Marshal(&a)
//	//log.Println(string(ja), ja, err)
//
//	log.Println("Can I access Config ? ", Config)
//}