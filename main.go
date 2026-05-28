package main

import (
	"go-star/core"
	"go-star/flags"
)

func main() {
	flags.Parse()
	core.ReadConf()
}
