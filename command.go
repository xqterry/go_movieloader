package main

type SendCommand struct {
	Code int `yaml:"code"`
	Names []string	`yaml:"names"`
	Index int		`yaml:"index"`
	Count int		`yaml:"count"`
	Width int		`yaml:"width"`
	Height int		`yaml:"height"`
	Movie string		`yaml:"movie"`
	CropW int		`yaml:"cropw"`
	CropH int		`yaml:"croph"`
	Group int		`yaml:"group"`
	Scale bool		`yaml:"scale"`
	CenterCrop bool		`yaml:"centercrop"`
	UseIFrame bool		`yaml:"iframe"`
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
