package main

import (
	"github.com/deepankarm/godantic/tools/godanticlint"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(godanticlint.Analyzer)
}
