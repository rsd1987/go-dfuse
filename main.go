package main

import (
	"fmt"
	"os"
	"time"

	"github.com/willtoth/go-dfuse/dfudevice"
	"github.com/willtoth/go-dfuse/dfufile"
	"gopkg.in/cheggaaa/pb.v1"
)

const (
	SPARKMAXDFUVID = 0x0483
	SPARKMAXDFUPID = 0xdf11
)

type consoleProgress struct {
	pb  *pb.ProgressBar
	inc uint
	max uint
}

func (c *consoleProgress) Reset() {
	c.pb.Reset(int(c.max))
	c.pb.Set(0)
	c.pb.Update()
	c.pb.Start()
}

func (c *consoleProgress) Increment() {
	c.pb.Add(int(c.inc))
	c.pb.Update()
}

func (c *consoleProgress) SetStatus(status string) {
	c.pb.Prefix(status)
}

func (c *consoleProgress) SetIncrement(increment uint) {
	c.inc = increment
}

func (c *consoleProgress) SetMax(max uint) {
	c.pb.SetTotal(int(max))
	c.max = max
}

func StartNew() consoleProgress {
	var c consoleProgress
	c.pb = pb.New(1)
	c.pb.SetMaxWidth(120)
	c.pb.ShowTimeLeft = false

	//Manually update the progress bar
	c.pb.SetRefreshRate(time.Second * 10000)
	return c
}

func main() {
	deviceList := dfudevice.List()
	for _, dev := range deviceList {
		fmt.Println(dev)
	}

	if len(deviceList) == 0 {
		return
	}

	var filename string
	var path string
	if len(os.Args) < 2 {
		return
	} else if len(os.Args) == 2 {
		if len(deviceList) > 1 {
			fmt.Println("Usage: go-dfuse.exe <path> <dfuFile>")
			fmt.Println("More than one device detected, must specify path")
			return
		}
		filename = os.Args[1]
		path = deviceList[0]
	} else {
		path = os.Args[1]
		filename = os.Args[2]
	}

	fmt.Println("Opening device...")

	dev, err := dfudevice.Open(path)
	defer dev.Close()

	if err != nil {
		fmt.Println("Failed to initialize ", err)
		return
	}

	bar := StartNew()

	dev.RegisterProgress(&bar)

	fmt.Println("Deviced Opened, reading ", filename)

	dfu, err := dfufile.Read(filename)

	if err != nil {
		fmt.Println("DFU File Format Failed: ", err)
		return
	}

	err = dfudevice.WriteImage(dfu.Images[0], dev)

	if err != nil {
		fmt.Println("Write DFUFile Failed ", err)
		return
	}

	verify, err := dfudevice.VerifyImage(dfu.Images[0], dev)

	if err != nil || verify == false {
		fmt.Println("Failed to verify DFU Image: ", err)
		return
	}

	err = dev.ExitDFU(uint(dfu.Images[0].Targets[0].Prefix.Address))

	if err != nil || verify == false {
		fmt.Println("Failed to exit DFU mode: ", err)
	}

	fmt.Println("")
	fmt.Println("Success!")
}
