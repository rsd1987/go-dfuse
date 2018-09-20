package dfudevice

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/gousb"
	"github.com/google/gousb/usbid"
)

const (
	cmdDETACH    = 0x00
	cmdDNLOAD    = 0x01
	cmdUPLOAD    = 0x02
	cmdGETSTATUS = 0x03
	cmdCLRSTATUS = 0x04
	cmdGETSTATE  = 0x05
	cmdABORT     = 0x06
)

const (
	statusOK          = 0x00
	statusErrorTarget = 0x01
	statusErrorFile   = 0x02
)

func ListDevices() {
	for _, dev := range GetDevices() {
		desc := dev.Desc
		fmt.Printf("%03d.%03d %s:%s %s\n", desc.Bus, desc.Address, desc.Vendor, desc.Product, usbid.Describe(desc))
	}
}

func GetDevices() []*gousb.Device {
	ctx := gousb.NewContext()
	defer ctx.Close()

	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		for _, cfg := range desc.Configs {
			for _, intf := range cfg.Interfaces {
				for _, ifSetting := range intf.AltSettings {
					if strings.Contains(usbid.Classify(ifSetting), "Device Firmware Update") ||
						strings.Contains(usbid.Classify(ifSetting), "DFU") {
						//fmt.Printf("%03d.%03d %s:%s %s\n", desc.Bus, desc.Address, desc.Vendor, desc.Product, usbid.Describe(desc))
						return true
					}
				}
			}
		}
		return false
	})

	// All Devices returned from OpenDevices must be closed.
	defer func() {
		for _, d := range devs {
			d.Close()
		}
	}()

	// OpenDevices can occaionally fail, so be sure to check its return value.
	if err != nil {
		log.Fatalf("list: %s", err)
	}

	return devs
}

func Test() {
	// Initialize a new Context.
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Open any device with a given VID/PID using a convenience function.
	dev, err := ctx.OpenDeviceWithVIDPID(0x0483, 0xdf11)
	defer dev.Close()
	if err != nil {
		log.Fatalf("Could not open a device: %v", err)
	}

	//dev.Control(gousb.ControlClass,)

	/*
		intf, done, err := dev.DefaultInterface()
		if err != nil {
			log.Fatalf("%s.DefaultInterface(): %v", dev, err)
		}
		defer done()
	*/
}

/*
func (d DfuDevice) Open() {

}

func (d DfuDevice) Close() {

}

func (d DfuDevice) Detatch() {

}
*/
