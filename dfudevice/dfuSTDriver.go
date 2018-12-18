// +build windows

package dfudevice

import (
	"fmt"

	"github.com/willtoth/go-STTub30"
	"github.com/willtoth/setupapi"
)

type dfuSTDriver struct {
	*sttub30.STDevice
}

func init() {
	var d dfuSTDriver
	addDriver(&d)
}

func (d dfuSTDriver) Open(vid, pid uint16) (dfuDevice DFUDevice, err error) {
	//GUID of STM32F3 DFU Driver
	guid := setupapi.Guid{0x3fe809ab, 0xfb91, 0x4cb5, [8]byte{0xa6, 0x43, 0x69, 0x67, 0x0d, 0x52, 0x36, 0x6e}}
	devInfo, err := setupapi.SetupDiGetClassDevsEx(guid, "", 0, setupapi.Present|setupapi.InterfaceDevice, 0, "", 0)
	if err != nil {
		return
	}

	devPath, err := devInfo.DevicePath(guid)
	if err != nil {
		return
	}

	dev, err := sttub30.Open(devPath)
	if err != nil {
		return
	}

	err = dev.SelectCurrentConfiguration(0, 0, 0)
	if err != nil {
		return
	}

	d = dfuSTDriver{&dev}
	dfuDevice.dev = d
	return
}

func (d dfuSTDriver) Control(rType, request uint8, val, idx uint16, data []byte) (int, error) {
	var req sttub30.ControlPipeRequest

	if rType&0x80 == 0 {
		req.Direction = sttub30.VendorDirectionOut
	} else {
		req.Direction = sttub30.VendorDirectionIn
	}

	//TODO: This should match rType
	req.Function = sttub30.UrbClassInterface
	req.Request = request
	req.Value = val
	req.Index = idx
	req.Length = uint64(len(data))

	err := d.ControlPipeRequest(req, data)

	if err != nil {
		return 0, err
	}

	return len(data), err
}

func (d dfuSTDriver) InterfaceDescription(cfgNum, intfNum, altNum int) (string, error) {
	//val, err := d.GetInterfaceDescriptor(uint(cfgNum), uint(intfNum), uint(altNum))
	rawDesc, err := d.GetInterfaceDescriptor(uint(0), uint(0), uint(0))

	if err != nil {
		return "", fmt.Errorf("Error getting interface descriptor: %v", err)
	}

	val, err := d.GetStringDescriptor(uint(rawDesc.InterfaceStringIndex))
	if err != nil {
		return "", fmt.Errorf("Error getting string for interface: %v", err)
	}
	return val, err
}

func (d dfuSTDriver) Close() {
	d.STDevice.Close()
}

func (d dfuSTDriver) List(VID, PID uint) []string {
	//GUID of STM32F3 DFU Driver
	guid := setupapi.Guid{0x3fe809ab, 0xfb91, 0x4cb5, [8]byte{0xa6, 0x43, 0x69, 0x67, 0x0d, 0x52, 0x36, 0x6e}}
	devInfo, err := setupapi.SetupDiGetClassDevsEx(guid, "", 0, setupapi.Present|setupapi.InterfaceDevice, 0, "", 0)
	if err != nil {
		return nil
	}

	devPath, err := devInfo.DevicePath(guid)
	if err != nil {
		return nil
	}

	dev, err := sttub30.Open(devPath)
	if err != nil {
		return nil
	}
	defer dev.Close()

	return nil
}
