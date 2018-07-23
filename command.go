package main

type SendCommand struct {
	Code int `yaml:"code"`
	Names []string	`yaml:"names"`
	Index int		`yaml:"index"`

}

type RecvCommand struct {
	Code int `yaml:"code"`
	Size int		`yaml:"size"`
}

/*
  commadn YAML
---
names: [movie1, movie2]
index: [0-9]+

# example:

names:
- aa
- bb
index: 3


 */

//func main() {
//	c := SendCommand{
//		Names: []string{"aa", "bb"},
//		//Index: 3,
//	}
//
//	str, err := yaml.Marshal(&c)
//	if err != nil{
//		log.Println("err", err)
//	} else {
//		log.Println(string(str))
//	}
//}