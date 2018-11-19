package main

import (
	"os"

	"github.com/willtoth/go-dfu/dfudevice"
)

func main() {
	dfudevice.Test(os.Args[1])
}
