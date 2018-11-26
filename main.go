package main

import (
	"fmt"
	"log"
	"os"

	"github.com/willtoth/go-dfu/dfudevice"
	"github.com/willtoth/go-dfu/dfufile"
)

const (
	SPARKMAXDFUVID = 0x0483
	SPARKMAXDFUPID = 0xdf11
)

func main() {
	filename := os.Args[1]

	dfuDevice, err := dfudevice.Open(SPARKMAXDFUVID, SPARKMAXDFUPID)
	defer dfuDevice.Close()

	if err != nil {
		log.Fatalf("Failed to initialize ", err)
	}

	dfu, err := dfufile.Read(filename)

	if err != nil {
		fmt.Println("DFU File Format Failed: ", err)
	}

	err = dfudevice.WriteDFUImage(dfu.Images[0], dfuDevice)

	if err != nil {
		fmt.Println("Write DFUFile Failed ", err)
	}
}
