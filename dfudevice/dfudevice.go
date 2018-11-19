package dfudevice

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
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

const (
	SPARKMAXDFUVID = 0x0483
	SPARKMAXDFUPID = 0xdf11
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

func Init() (err error) {
	// Initialize a new Context.
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Open any device with a given VID/PID using a convenience function.
	var found bool
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if found {
			return false
		}
		if desc.Vendor == SPARKMAXDFUVID && desc.Product == SPARKMAXDFUPID {
			found = true
			return true
		}
		return false
	})
	if len(devs) == 0 {
		return fmt.Errorf("No DFU Device Found")
	}

	defer func() {
		for _, dev := range devs {
			dev.Close()
		}
	}()

	return nil
}

func clearStatus() {

}

func getStatus() {

}

func massErase() {

}

func pageErase() {

}

func setAddress() {

}

func writeMemory() {

}

func writePage() {

}

func exitDFU() {

}

func computeCRC() {

}

func Test(filename string) {
	_, err := ReadDFUFile(filename)

	if err != nil {
		fmt.Println("DFU File Format Failed: %v", err)
	}
}

type DFUTarget struct {
	Prefix struct {
		Address uint32
		Size    uint32
	}
	Elements []byte
}

type DFUImage struct {
	Prefix struct {
		Signature  [6]byte
		AltSetting bool
		IsNamed    uint32
		Name       [255]byte
		Size       uint32
		Elements   uint32
	}
	Targets []DFUTarget
}

type DFUFile struct {
	Prefix struct {
		Signature [5]byte
		Version   uint8
		Size      uint32
		Targets   uint8
	}

	Images []DFUImage

	Suffix struct {
		DeviceVersion uint16
		Product       uint16
		Vendor        uint16
		DfuFormat     uint16
		Ufd           [3]byte
		Length        uint8
		Crc32         uint32
	}
}

func ReadDFUFile(filename string) (DFUFile, error) {
	var fileData DFUFile

	fileHandle, err := os.Open(filename)
	defer fileHandle.Close()

	//TODO: Verbose output
	//fmt.Println(filename)

	if err != nil {
		return fileData, err
	}

	//   <   little endian
	//   5s  char[5]     signature   "DfuSe"
	//   B   uint8_t     version     1
	//   I   uint32_t    size        Size of the DFU file (not including suffix)
	//   B   uint8_t     targets     Number of targets
	err = binary.Read(fileHandle, binary.LittleEndian, &fileData.Prefix)

	if err != nil {
		return fileData, err
	}

	if string(fileData.Prefix.Signature[:]) != "DfuSe" {
		return fileData, fmt.Errorf("Error in image prefix, dfu file failed")
	}

	//TODO: Verbose output
	//fmt.Printf("Signature: %x, v%d, image size: %d, targets: %d\r\n",
	//	fileData.Prefix.Signature,
	//	fileData.Prefix.Version,
	//	fileData.Prefix.Size,
	//	fileData.Prefix.Targets)

	fileData.Images = make([]DFUImage, fileData.Prefix.Targets)

	for imageIdx := range fileData.Images {

		image := &fileData.Images[imageIdx]

		// Decode the Image Prefix
		//   <   little endian
		//   6s      char[6]     signature   "Target"
		//   B       uint8_t     altsetting
		//   I       uint32_t    named       bool indicating if a name was used
		//   255s    char[255]   name        name of the target
		//   I       uint32_t    size        size of image (not incl prefix)
		//   I       uint32_t    elements    Number of elements in the image
		err = binary.Read(fileHandle, binary.LittleEndian, &image.Prefix)

		if err != nil {
			return fileData, err
		}

		if string(image.Prefix.Signature[:]) != "Target" {
			return fileData, fmt.Errorf("Error in image prefix, dfu file failed")
		}

		//TODO: Verbose output
		//fmt.Printf("Signature: %x, num: %d, alt settings: %d, name: %s, size: %d, elements: %d\r\n",
		//	image.Prefix.Signature,
		//	image.Prefix.IsNamed,
		//	image.Prefix.AltSetting,
		//	image.Prefix.Name,
		//	image.Prefix.Size,
		//	image.Prefix.Elements)

		image.Targets = make([]DFUTarget, image.Prefix.Elements)

		for targetIdx := range image.Targets {

			// Decode target prefix
			//   <   little endian
			//   I   uint32_t    element address
			//   I   uint32_t    element size
			err = binary.Read(fileHandle, binary.LittleEndian, &image.Targets[targetIdx].Prefix)
			if err != nil {
				return fileData, err
			}

			image.Targets[targetIdx].Elements = make([]byte, image.Targets[targetIdx].Prefix.Size)
			_, err = fileHandle.Read(image.Targets[targetIdx].Elements)

			if err != nil {
				return fileData, err
			}
		}
	}

	//   <   little endian
	//   H   uint16_t    device  Firmware version
	//   H   uint16_t    product
	//   H   uint16_t    vendor
	//   H   uint16_t    dfu     0x11a   (DFU file format version)
	//   3s  char[3]     ufd     'UFD'
	//   B   uint8_t     len     16
	//   I   uint32_t    crc32
	err = binary.Read(fileHandle, binary.LittleEndian, &fileData.Suffix)

	//TODO: Verbose output
	//fmt.Printf("version: %x, product: %04x, vendor: %04x, format: %d, length: %d, crc32: %d\r\n",
	//	fileData.Suffix.DeviceVersion,
	//	fileData.Suffix.Product,
	//	fileData.Suffix.Vendor,
	//	fileData.Suffix.DfuFormat,
	//	fileData.Suffix.Length,
	//	fileData.Suffix.Crc32)

	if string(fileData.Suffix.Ufd[:]) != "UFD" {
		return fileData, fmt.Errorf("Error in suffix prefix, dfu file failed")
	}

	//TODO: check CRC

	return fileData, nil
}

/*
func (d DfuDevice) Open() {

}

func (d DfuDevice) Close() {

}

func (d DfuDevice) Detatch() {

}
*/
