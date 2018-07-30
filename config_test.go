package main

import (
	"testing"
	//"log"

	"regexp"
	"math/rand"
	"time"
)

func TestParseConfig(t *testing.T) {
	conf := NewMoviesConfig("./config/config.yaml")
	//t.Log("conf loaded", conf)

	c := conf.FindMovieFileConfigByName("home_movie3")
	t.Log("conf movie1 ", c)

	rand.Seed(time.Now().Unix())
	ss := c.Skip[rand.Intn(len(c.Skip))]
	t.Log("rand ", ss)
}

func TestReg(t *testing.T){
	p := "^anim*"
	ss := []string{"anim1", "anim2", "movie1", "home_movie1", "home_anim1"}

	for _, s := range(ss) {
		m, err := regexp.MatchString(p, s)
		t.Log("Matched ", m, err)
	}

	conf := NewMoviesConfig("./config/config.yaml")
	slice := conf.FindMatchedConfigs("^mov")

	t.Log("before shuffle", slice)
	for _, s := range(slice){
		t.Log(s.Name)
	}

	rand.Seed(time.Now().Unix())
	for i := len(slice) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}

	t.Log("after shuffle")
	count := 10000
	pc := count
	if len(slice) > 0 {
		pc = count / len(slice)
	}
	sum := 0
	mod := count % pc
	var cc []int
	for _, s := range(slice) {
		sum += pc
		t.Log(s.Name, pc, sum)
		cc = append(cc, pc)
	}

	if mod != 0 {
		cc[0] += mod
	}

	t.Log(cc)
	sum = 0
	for _, c := range(cc) {
		sum += c
	}

	if sum != count {
		t.Fatal("not same")
	}
}
