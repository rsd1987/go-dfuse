package dfudevice

import (
	"fmt"
	"log"

	"github.com/google/gousb"
)

type dfulibusb struct {
	*gousb.Device
	ctx *gousb.Context
}

func init() {
	// TODO: Change list string to return the value for Open()
	//var d dfulibusb
	//addDriver(&d)
}

func (d dfulibusb) Open(path string) (dfuDevice DFUDevice, err error) {
	var vid, pid uint16
	// Initialize a new Context.
	ctx := gousb.NewContext()

	// Open any device with a given VID/PID using a convenience function.
	var found bool
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if found {
			return false
		}
		if desc.Vendor == gousb.ID(vid) && desc.Product == gousb.ID(pid) {
			found = true
			return true
		}
		return false
	})

	//TODO: sometimes this throws an unrelated error even if its not related to this particular device
	//ignore for now as another call below will error if there is truly an error
	//if err != nil {
	//	return err
	//}

	if len(devs) == 0 {
		err = fmt.Errorf("No DFU Device Found")
		return
	} else if len(devs) > 1 {
		err = fmt.Errorf("More than 1 DFU device found")
		return
	}

	device := &dfulibusb{devs[0], ctx}
	device.ControlTimeout = 5000000000 //5s

	//TODO: This should find the correct interface if possible
	// Claim the default interface
	_, done, err := device.DefaultInterface()
	if err != nil {
		log.Fatalf("%s.DefaultInterface(): %v", device, err)
	}

	defer done()

	dfuDevice.dev = device
	err = dfuDevice.ClearStatus()

	return
}

func (d *dfulibusb) List() []string {
	var VID, PID uint
	devices := make([]string, 0)
	ctx := gousb.NewContext()
	defer ctx.Close()

	devs, _ := ctx.OpenDevices(func(d *gousb.DeviceDesc) bool {
		if (d.Vendor == gousb.ID(VID)) && (d.Product == gousb.ID(PID)) {
			devices = append(devices, fmt.Sprintf("DFU Device Bus: %d.%d ID: %s:%s", d.Bus, d.Address, d.Vendor, d.Product))
			return true
		}
		return false
	})

	// All Devices returned from OpenDevices must be closed.
	defer func() {
		for _, d := range devs {
			d.Close()
		}
	}()

	return devices
}

func (d *dfulibusb) Close() {
	if d != nil {
		d.Device.Close()
	}

	if d.ctx != nil {
		d.ctx.Close()
	}
}
