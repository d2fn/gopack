package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

const (
	GopackDir = ".gopack"
	VendorDir = ".gopack/vendor"
)

const (
	Blue     = uint8(94)
	Green    = uint8(92)
	Red      = uint8(31)
	Gray     = uint8(90)
	EndColor = "\033[0m"
)

var (
	pwd string
)

func main() {
	fmtcolor(104, "/// g o p a c k ///")
	fmt.Println()
	setupEnv()
	d := LoadDependencyModel()
	// prepare dependencies
	d.VisitDeps(
		func(d *Dep) {
			fmtcolor(Gray, "updating %s\n", d.Import)
			d.goGetUpdate()
			fmtcolor(Gray, "pointing %s at %s %s\n", d.Import, d.CheckoutType(), d.CheckoutSpec)
			d.switchToBranchOrTag()
		})
	stats, err := Analyze(".")
	fmt.Print(stats)
	// run the specified command
	cmd := exec.Command("go", os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fail(err)
	}
	fmtcolor(Green, "done\n")
}

// set GOPATH to the local vendor dir
func setupEnv() {
	dir, err := os.Getwd()
	pwd = dir
	if err != nil {
		fail(err)
	}
	vendor := fmt.Sprintf("%s/%s", pwd, VendorDir)
	fmt.Println(pwd)
	err = os.Setenv("GOPATH", vendor)
	if err != nil {
		fail(err)
	}
}

func fmtcolor(c uint8, s string, args ...interface{}) {
	fmt.Printf("\033[%dm", c)
	if len(args) > 0 {
		fmt.Printf(s, args...)
	} else {
		fmt.Printf(s)
	}
	fmt.Printf("%s", EndColor)
}

func logcolor(c uint8, s string, args ...interface{}) {
	log.Printf("\033[%dm", c)
	if len(args) > 0 {
		log.Printf(s, args...)
	} else {
		log.Printf(s)
	}
	log.Printf("%s", EndColor)
}

func failf(s string, args ...interface{}) {
	fmtcolor(Red, s, args...)
	os.Exit(1)
}

func fail(a ...interface{}) {
	fmt.Printf("\033[%dm", Red)
	fmt.Print(a)
	fmt.Printf("%s", EndColor)
	os.Exit(1)
}
