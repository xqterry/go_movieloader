package main

import (
	"testing"
	//"log"

)

func TestParseConfig(t *testing.T) {
	conf := NewMoviesConfig("./config/config.yaml")
	//t.Log("conf loaded", conf)

	c := conf.FindMovieFileConfigByName("movie1")
	t.Log("conf movie1 ", c)
}