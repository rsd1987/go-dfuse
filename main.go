package main

import (
	"fmt"
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

	fmt.Println("Opening device...")

	dev, err := dfudevice.Open(SPARKMAXDFUVID, SPARKMAXDFUPID)
	defer dev.Close()

	if err != nil {
		fmt.Println("Failed to initialize ", err)
		return
	}

	fmt.Println("Deviced Opened, reading %s", filename)

	dfu, err := dfufile.Read(filename)

	if err != nil {
		fmt.Println("DFU File Format Failed: ", err)
		return
	}

	fmt.Println("Writing Image...")

	err = dfudevice.WriteImage(dfu.Images[0], dev)

	if err != nil {
		fmt.Println("Write DFUFile Failed ", err)
		return
	}

	fmt.Println("Verifying Image...")

	verify, err := dfudevice.VerifyImage(dfu.Images[0], dev)

	if err != nil || verify == false {
		fmt.Println("Failed to verify DFU Image: ", err)
		return
	}

	fmt.Println("Leaving DFU Mode...")

	err = dev.ExitDFU(uint(dfu.Images[0].Targets[0].Prefix.Address))

	if err != nil || verify == false {
		fmt.Println("Failed to exit DFU mode: ", err)
	}
}
