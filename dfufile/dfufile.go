package dfufile

import (
	"encoding/binary"
	"fmt"
	"os"
)

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

func Read(filename string) (DFUFile, error) {
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
