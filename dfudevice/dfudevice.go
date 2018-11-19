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
	readDFUFile(filename)
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
		Ufd           string
		Length        uint8
		Crc32         uint32
	}
}

func readDFUFile(filename string) (DFUFile, error) {
	var fileData DFUFile

	fileHandle, err := os.Open(filename)
	defer fileHandle.Close()

	fmt.Println(filename)

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
		fmt.Printf("Error: %v\r\n", err)
		return fileData, err
	}

	fmt.Printf("Signature: %x, v%d, image size: %d, targets: %d\r\n",
		fileData.Prefix.Signature,
		fileData.Prefix.Version,
		fileData.Prefix.Size,
		fileData.Prefix.Targets)

	for targets := uint8(0); targets < fileData.Prefix.Targets; targets++ {
		// Decode the Image Prefix
		//   <   little endian
		//   6s      char[6]     signature   "Target"
		//   B       uint8_t     altsetting
		//   I       uint32_t    named       bool indicating if a name was used
		//   255s    char[255]   name        name of the target
		//   I       uint32_t    size        size of image (not incl prefix)
		//   I       uint32_t    elements    Number of elements in the image
		var image DFUImage
		err = binary.Read(fileHandle, binary.LittleEndian, &image.Prefix)

		if err != nil {
			fmt.Printf("Error: %v\r\n", err)
			return fileData, err
		}

		fmt.Printf("Signature: %x, num: %d, alt settings: %d, name: %s, size: %d, elements: %d\r\n",
			image.Prefix.Signature,
			image.Prefix.IsNamed,
			image.Prefix.AltSetting,
			image.Prefix.Name,
			image.Prefix.Size,
			image.Prefix.Elements)

		image.Targets = make([]DFUTarget, image.Prefix.Elements)

		for idx := range image.Targets {

			// Decode target prefix
			//   <   little endian
			//   I   uint32_t    element address
			//   I   uint32_t    element size
			err = binary.Read(fileHandle, binary.LittleEndian, &image.Targets[idx].Prefix)
			if err != nil {
				fmt.Printf("Error: %v\r\n", err)
				return fileData, err
			}

			image.Targets[idx].Elements = make([]byte, image.Targets[idx].Prefix.Size)
			_, err = fileHandle.Read(image.Targets[idx].Elements)

			if err != nil {
				fmt.Printf("Error: %v\r\n", err)
				return fileData, err
			}
		}
	}

	return fileData, nil

	/*
		    for target_idx in range(dfu_prefix['targets']):
				// Decode the Image Prefix

		        // <6sBI255s2I
		        //   <   little endian
		        //   6s      char[6]     signature   "Target"
		        //   B       uint8_t     altsetting
		        //   I       uint32_t    named       bool indicating if a name was used
		        //   255s    char[255]   name        name of the target
		        //   I       uint32_t    size        size of image (not incl prefix)
		        //   I       uint32_t    elements    Number of elements in the image
		        img_prefix, data = consume('<6sBI255s2I', data,
		                                   'signature altsetting named name '
		                                   'size elements')
		        img_prefix['num'] = target_idx
		        if img_prefix['named']:
		            img_prefix['name'] = cstring(img_prefix['name'])
		        else:
		            img_prefix['name'] = ''
		        print('    %(signature)s %(num)d, alt setting: %(altsetting)s, '
		              'name: "%(name)s", size: %(size)d, elements: %(elements)d'
		              % img_prefix)

		        target_size = img_prefix['size']
		        target_data, data = data[:target_size], data[target_size:]
		        for elem_idx in range(img_prefix['elements']):
		            // Decode target prefix
		            //   <   little endian
		            //   I   uint32_t    element address
		            //   I   uint32_t    element size
		            elem_prefix, target_data = consume('<2I', target_data, 'addr size')
		            elem_prefix['num'] = elem_idx
		            print('      %(num)d, address: 0x%(addr)08x, size: %(size)d'
		                  % elem_prefix)
		            elem_size = elem_prefix['size']
		            elem_data = target_data[:elem_size]
		            target_data = target_data[elem_size:]
		            elem_prefix['data'] = elem_data
		            elements.append(elem_prefix)

		        if len(target_data):
		            print("target %d PARSE ERROR" % target_idx)

		    // Decode DFU Suffix
		    //   <   little endian
		    //   H   uint16_t    device  Firmware version
		    //   H   uint16_t    product
		    //   H   uint16_t    vendor
		    //   H   uint16_t    dfu     0x11a   (DFU file format version)
		    //   3s  char[3]     ufd     'UFD'
		    //   B   uint8_t     len     16
		    //   I   uint32_t    crc32
		    dfu_suffix = named(struct.unpack('<4H3sBI', data[:16]),
		                       'device product vendor dfu ufd len crc')
		    print ('    usb: %(vendor)04x:%(product)04x, device: 0x%(device)04x, '
		           'dfu: 0x%(dfu)04x, %(ufd)s, %(len)d, 0x%(crc)08x' % dfu_suffix)
		    if crc != dfu_suffix['crc']:
		        print("CRC ERROR: computed crc32 is 0x%08x" % crc)
		        return
		    data = data[16:]
		    if data:
		        print("PARSE ERROR")
		        return

		    return elements
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
