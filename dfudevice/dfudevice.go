package dfudevice

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
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

const dfuINTERFACE = 0

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
		if desc.Vendor == SPARKMAXDFUVID && desc.Product == SPARKMAXDFUPID {
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

	// OpenDevices can occaionally fail, so be sure to check its return value.
	if err != nil {
		log.Fatalf("list: %s", err)
	}

	return devs
}

type DFUDevice struct {
	ctx *gousb.Context
	dev *gousb.Device
}

func (d DFUDevice) Close() {
	if d.dev != nil {
		d.dev.Close()
	}

	if d.ctx != nil {
		d.ctx.Close()
	}
}

func Init() (device DFUDevice, err error) {
	// Initialize a new Context.
	device.ctx = gousb.NewContext()

	ctx := device.ctx

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

	//TODO: sometimes this throws an unrelated error even if its not related to this particular device
	//ignore for now as another call below will error if there is truly an error
	//if err != nil {
	//	return err
	//}

	if len(devs) == 0 {
		return device, fmt.Errorf("No DFU Device Found")
	} else if len(devs) > 1 {
		return device, fmt.Errorf("More than 1 DFU device found")
	}

	device.dev = devs[0]

	//TODO: This should find the correct interface if possible
	// Claim the default interface
	_, done, err := device.getInterface()
	if err != nil {
		log.Fatalf("%s.DefaultInterface(): %v", device.dev, err)
	}
	defer done()

	return device, device.ClearStatus()
}

func (d DFUDevice) getInterface() (intf *gousb.Interface, done func(), err error) {
	return d.dev.DefaultInterface()
}

func (d DFUDevice) ClearStatus() error {
	if d.dev == nil {
		return fmt.Errorf("ClearStatus(): Device not initialized")
	}

	_, err := d.dev.Control(0x21, cmdCLRSTATUS, 0, dfuINTERFACE, nil)

	return err
}

func (d DFUDevice) GetStatus() (byte, error) {
	if d.dev == nil {
		return 0, fmt.Errorf("GetStatus(): Device not initialized")
	}

	var dfuStatus [32]byte

	_, err := d.dev.Control(0xA1, cmdGETSTATUS, 0, dfuINTERFACE, dfuStatus[:])

	return dfuStatus[4], err
}

func (d DFUDevice) MassErase() error {
	if d.dev == nil {
		return fmt.Errorf("MassErase(): Device not initialized")
	}

	_, err := d.dev.Control(0x21, cmdDNLOAD, 0, dfuINTERFACE, []byte{0x41})

	return err
}

func (d DFUDevice) PageErase(page uint) error {
	return nil
}

func setAddress() {

}

func (d DFUDevice) WriteMemory(addr uint, data []byte) error {
	return nil
}

func writePage() {

}

func exitDFU() {

}

func computeCRC() {

}

type MemoryLayout struct {
	StartAddress uint
	Pages        uint
	PageSize     uint
	Size         uint
}

func (d DFUDevice) GetMemoryLayout() (mem []MemoryLayout, err error) {
	//intf, done, _ := d.getInterface()
	//defer done()

	//TODO: Get the proper config and interface id
	desc, err := d.dev.InterfaceDescription(1, 0, 0)

	descValues := strings.Split(desc, "/")

	addr, err := strconv.ParseUint(descValues[1], 0, 32)
	segments := strings.Split(descValues[2], ",")

	segmentRegex := regexp.MustCompile(`(\d+)\*(\d+)(.)(.)`)

	mem = make([]MemoryLayout, len(segments))

	for idx, segment := range segments {
		segMatches := segmentRegex.FindAllStringSubmatch(segment, 3)

		if segMatches == nil || len(segMatches[0]) < 3 {
			err = fmt.Errorf("Bad descriptor returned from usb device, unable to parse memory map")
			return mem, err
		}

		numPages, err := strconv.ParseUint(segMatches[0][1], 0, 32)
		if err != nil {
			continue
		}
		pageSize, err := strconv.ParseUint(segMatches[0][2], 0, 32)
		if err != nil {
			continue
		}
		multiplier := segMatches[0][3]

		if multiplier[0] == 'K' {
			pageSize *= 1024
		} else if multiplier[0] == 'M' {
			pageSize *= 1024 * 1024
		}

		mem[idx].StartAddress = uint(addr)
		mem[idx].Pages = uint(numPages)
		mem[idx].PageSize = uint(pageSize)
		mem[idx].Size = mem[idx].Pages * mem[idx].PageSize

		addr += uint64(mem[idx].Size)
	}
	return
}

func WriteDFUImage(dfuImage DFUImage, dfuDevice DFUDevice) error {
	massErase := false

	mem, err := dfuDevice.GetMemoryLayout()

	//TODO: This should search mem[] for the correct location
	memory := mem[0]

	//Check that target fits within mem
	//if uint(dfuImage.Prefix.Address+dfuTarget.Prefix.Size) > mem[0].StartAddress+mem[0].Size {
	//	return fmt.Errorf("Target address of %x and size of %d will not fit within specified device.",
	//		dfuTarget.Prefix.Address, dfuTarget.Prefix.Size)
	//}

	//fmt.Printf("Writing to device starting at page 0x%x\r\n", dfuTarget.Prefix.Address)

	fmt.Println("Erasing pages...")

	if massErase == true {
		err = dfuDevice.MassErase()

		if err != nil {
			return err
		}
	} else {
		for _, target := range dfuImage.Targets {
			startPage := -1
			pagesToErase := uint(math.Ceil(float64(target.Prefix.Size) / float64(memory.PageSize)))

			if int(target.Prefix.Size) != len(target.Elements) {
				return fmt.Errorf("Mismatch target size, size claims %d, but has %d elements", target.Prefix.Size, len(target.Elements))
			}

			for idx := uint(0); idx < memory.Pages; idx++ {
				//Target should be at page boundary
				if memory.StartAddress+(idx*memory.PageSize) == uint(target.Prefix.Address) {
					startPage = int(idx)
					break
				}
			}

			if startPage == -1 {
				return fmt.Errorf("Failed to find target address %x in device memory", target.Prefix.Address)
			} else {
				for numPages := 0; numPages < int(pagesToErase); numPages++ {
					err = dfuDevice.PageErase(uint(startPage + numPages))

					if err != nil {
						return err
					}
				}
			}
		}
	}

	//By this point, the appropriate amount of flash has been erased, write each target
	for _, target := range dfuImage.Targets {
		dfuDevice.WriteMemory(uint(target.Prefix.Address), target.Elements)
	}

	return err
}

func Test(filename string) {
	//ListDevices()

	dfuDevice, err := Init()
	defer dfuDevice.Close()

	if err != nil {
		log.Fatalf("Failed to initialize ", err)
	}

	dfu, err := ReadDFUFile(filename)

	if err != nil {
		fmt.Println("DFU File Format Failed: ", err)
	}

	err = WriteDFUImage(dfu.Images[0], dfuDevice)

	if err != nil {
		fmt.Println("Write DFUFile Failed ", err)
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
