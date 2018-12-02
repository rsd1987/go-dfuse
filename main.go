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

	dev, err := dfudevice.Open(SPARKMAXDFUVID, SPARKMAXDFUPID)
	defer dev.Close()

	if err != nil {
		log.Fatalf("Failed to initialize ", err)
	}

	dfu, err := dfufile.Read(filename)

	if err != nil {
		fmt.Println("DFU File Format Failed: ", err)
	}

	err = dfudevice.WriteDFUImage(dfu.Images[0], dev)

	if err != nil {
		fmt.Println("Write DFUFile Failed ", err)
	}

	data, err := dev.ReadMemory(0x08000000, 2048*5+1000)

	if err != nil {
		fmt.Println("Read Memory Failed: %v", err)
	} else {
		fmt.Println(len(data))
	}
}
