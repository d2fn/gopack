package main

import (
	"log"
	"os"
	"os/exec"
	"fmt"
)

const (
	VendorDir = "vendor"
)

var (
	pwd string
)

func main() {
	setupEnv()
	log.Println(pwd)
	d := LoadDependencyModel()
	log.Println(d.String())
	// prepare dependencies
	d.VisitDeps(
		func (d *Dep) {
			log.Printf("updating dependency %s\n", d.Import)
			d.goGetUpdate()
			log.Printf("pointing %s at %s\n", d.Import, d.checkoutName())
			d.switchToBranchOrTag()
		})
	// run the specified command
	cmd := exec.Command("go", os.Args[1:]...)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("done")
}

// set GOPATH to the local vendor dir
func setupEnv() {
	// build the go command to fork once we've made sure dependencies are in place
	dir, err := os.Getwd()
	pwd = dir
	log.Println(pwd)
	if err != nil {
		log.Fatal(err)
	}
	vendor := fmt.Sprintf("%s/%s", pwd, VendorDir)
	err = os.Setenv("GOPATH", vendor)
	if err != nil {
		log.Fatal(err)
	}
}

