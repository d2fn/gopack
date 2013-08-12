package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestSetPwdDefault(t *testing.T) {
	setPwd()
	dir, _ := os.Getwd()
	if pwd != dir {
		t.Errorf("Expected pwd to be %s but it was %s.\n", dir, pwd)
	}
}

func TestSetPwdAppConfig(t *testing.T) {
	dir, _ := ioutil.TempDir("", "gopack-test-")
	os.Setenv("GOPACK_APP_CONFIG", dir)
	setPwd()
	if pwd != dir {
		t.Errorf("Expected pwd to be %s but it was %s.\n", dir, pwd)
	}
}
