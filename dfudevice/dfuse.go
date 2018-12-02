package dfudevice

import (
	"bytes"
	"fmt"
	"math"

	"github.com/willtoth/go-dfu/dfufile"
)

func WriteImage(dfuImage dfufile.DFUImage, dfuDevice DFUDevice) error {
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

	//fmt.Println("Erasing pages...")

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
					err = dfuDevice.PageErase(uint(target.Prefix.Address) + (uint(startPage+numPages) * memory.PageSize))

					if err != nil {
						return err
					}
				}
			}
		}
	}

	//fmt.Println("Writing pages...")

	//By this point, the appropriate amount of flash has been erased, write each target
	for _, target := range dfuImage.Targets {
		//fmt.Printf("Writing to address 0x%x\r\n", target.Prefix.Address)
		dfuDevice.WriteMemory(uint(target.Prefix.Address), target.Elements)
	}

	return err
}

func VerifyImage(dfuImage dfufile.DFUImage, dfuDevice DFUDevice) (bool, error) {
	for _, target := range dfuImage.Targets {
		deviceData, err := dfuDevice.ReadMemory(uint(target.Prefix.Address), uint(target.Prefix.Size))

		if err != nil {
			return false, fmt.Errorf("Verify failed to read device memory: %v", err)
		}

		if bytes.Equal(deviceData, target.Elements) == false {
			return false, nil
		}
	}
	return true, nil
}
