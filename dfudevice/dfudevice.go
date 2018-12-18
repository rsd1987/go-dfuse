package dfudevice

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//DFU Commands
const (
	cmdDETACH    = 0x00
	cmdDNLOAD    = 0x01
	cmdUPLOAD    = 0x02
	cmdGETSTATUS = 0x03
	cmdCLRSTATUS = 0x04
	cmdGETSTATE  = 0x05
	cmdABORT     = 0x06
)

//DFU States
const (
	dfuStateAppIdle              = 0x00
	dfuStateAppDetach            = 0x01
	dfuStateDfuIdle              = 0x02
	dfuStateDfuDownloadSync      = 0x03
	dfuStateDfuDownloadBusy      = 0x04
	dfuStateDfuDownloadIdle      = 0x05
	dfuStateDfuManifestSync      = 0x06
	dfuStateDfuManifest          = 0x07
	dfuStateDfuManifestWaitReset = 0x08
	dfuStateDfuUploadIdle        = 0x09
	dfuStateDfuError             = 0x0a
)

//DFU Status
const (
	dfuStatusOk               = 0x00
	dfuStatusErrorTarget      = 0x01
	dfuStatusErrorFile        = 0x02
	dfuStatusErrorWrite       = 0x03
	dfuStatusErrorErase       = 0x04
	dfuStatusErrorCheckErased = 0x05
	dfuStatusErrorProg        = 0x06
	dfuStatusErrorVerify      = 0x07
	dfuStatusErrorAddress     = 0x08
	dfuStatusErrorNotDone     = 0x09
	dfuStatusErrorFirmware    = 0x0a
	dfuStatusErrorVendor      = 0x0b
	dfuStatusErrorUsbr        = 0x0c
	dfuStatusErrorPor         = 0x0d
	dfuStatusErrorUnknown     = 0x0e
	dfuStatusErrorStalledPkt  = 0x0f
)

type dfuStatus struct {
	bStatus       uint8
	bwPollTimeout uint
	bState        uint8
	iString       uint8
}

func (d dfuStatus) String() string {
	return []string{
		"No error condition is present",
		"File is not targeted for use by this device",
		"File is for this device but fails some vendor-specific verification test",
		"Device is unable to write memory",
		"Memory erase function failed",
		"Memory erase check failed",
		"Program memory function failed",
		"Programmed memory failed verification",
		"Cannot program memory due to received address that is out of range",
		"Received DFU_DNLOAD with wLength = 0, but device does not think it has all of the data yet",
		"Device’s firmware is corrupt.  It cannot return to run-time (non-DFU) operations",
		"Vendor-specific error",
		"Device detected unexpected USB reset signaling",
		"Device detected unexpected power on reset",
		"Something went wrong, but the device does not know what it was",
		"Device stalled an unexpected request",
	}[d.iString]
}

func (d dfuStatus) Wait() {
	//Status command tells the correct polling timeout
	time.Sleep(time.Millisecond * time.Duration(d.bwPollTimeout))
}

const dfuINTERFACE = 0

func List(VID, PID uint) []string {
	result := make([]string, 0)
	for _, driver := range dfuDriverList {
		result = append(result, driver.List(VID, PID)...)
	}
	return result
}

type DFUDevice struct {
	dev dfuDriver

	progressBars progressList
}

func (d *DFUDevice) RegisterProgress(progress Progress) {
	d.progressBars.add(progress)
}

func (d DFUDevice) Close() {
	if d.dev != nil {
		d.dev.Close()
	}
}

func Open(vid, pid uint16) (device DFUDevice, err error) {
	//Return the first successful opened driver
	for _, driver := range dfuDriverList {
		device, err = driver.Open(vid, pid)
		if err == nil {
			return
		}
	}
	return
}

func (d DFUDevice) ClearStatus() error {
	if d.dev == nil {
		return fmt.Errorf("ClearStatus(): Device not initialized")
	}

	_, err := d.dev.Control(0x21, cmdCLRSTATUS, 0, dfuINTERFACE, nil)

	return err
}

func (d DFUDevice) GetStatus() (status dfuStatus, err error) {
	status.bStatus = dfuStatusErrorUnknown
	status.bState = dfuStateDfuError
	if d.dev == nil {
		err = fmt.Errorf("In GetStatus(): Device not initialized")
		return
	}

	var rawbuf [6]byte

	_, err = d.dev.Control(0xA1, cmdGETSTATUS, 0, dfuINTERFACE, rawbuf[:])

	if err != nil {
		err = fmt.Errorf("Control transfer in GetStatus() failed: %v", err)
		return
	}

	status.bStatus = rawbuf[0]
	status.bwPollTimeout = uint(rawbuf[1]) | uint(rawbuf[2])<<8 | uint(rawbuf[3])<<16
	status.bState = rawbuf[4]
	status.iString = rawbuf[5]

	//Wait the bwPollTimeout() time TODO: Should this be here or up to the implementation?
	status.Wait()

	return
}

const (
	dnloadCmdErase         = 0x41
	dnloadCmdReadUnprotect = 0x92 //not implemented
	dnloadCmdSetAddress    = 0x21
)

func (d DFUDevice) dnloadSpecialCommand(command byte, buffer []byte) error {
	cmdBuffer := make([]byte, len(buffer)+1)

	cmdBuffer[0] = command
	copy(cmdBuffer[1:], buffer)

	return d.dnloadCommand(0, cmdBuffer)
}

func (d DFUDevice) dnloadWaitOnIdle() error {
	var status dfuStatus
	var err error

	//TODO: Implement timeouts
	for true {
		//Check that the device is in the IDLE or DNLOAD-IDLE states before proceeding
		status, err = d.GetStatus()

		if err != nil {
			return fmt.Errorf("Initial GetStatus() failed in dnload command: %v: ", err)
		}

		if status.bState != dfuStateDfuIdle && status.bState != dfuStateDfuDownloadIdle {
			//fmt.Printf("State is not set for dnload, found to be: %d, string %s\r\nAttempting to clear status...\r\n", status.bState, status)
			d.ClearStatus()
		} else {
			break
		}
	}

	return err
}

//dnload requests implemented per STM32 app note AN3156
func (d DFUDevice) dnloadCommand(wValue uint16, buffer []byte) error {
	if d.dev == nil {
		return fmt.Errorf("dnloadSpecialCommand(): Device not initialized")
	}

	var status dfuStatus
	var err error

	err = d.dnloadWaitOnIdle()

	if err != nil {
		return err
	}

	_, err = d.dev.Control(0x21, cmdDNLOAD, wValue, dfuINTERFACE, buffer)

	if err != nil {
		return fmt.Errorf("Control Transfer failed after initial dnload command: %v: ", err)
	}

	status, err = d.GetStatus()

	if err != nil {
		return fmt.Errorf("Failed to get status after dnload command: %d, %v", err)
	}

	//First status should always return dfuStateDfuDownloadBusy, this starts the operation
	if status.bState != dfuStateDfuDownloadBusy {
		return fmt.Errorf("Wrong state after dnload command expected dfuStateDfuDownloadBusy")
	}

	//TODO: Add additional condition for timeout
	for status.bState == dfuStateDfuDownloadBusy {

		status, err = d.GetStatus()

		if err != nil {
			return fmt.Errorf("Failed while polling status during dnload command: %v", err)
		}

		//Handle unexpected states
		if status.bState != dfuStateDfuDownloadIdle && status.bState != dfuStateDfuDownloadBusy {
			errorString := fmt.Sprintf("Wrong state after dnl command ")
			if status.bState == dfuStateDfuError {
				switch bStatus := status.bStatus; bStatus {
				case dfuStatusErrorTarget:
					return fmt.Errorf("%s, wrong or unsupported page address, code string: %s", errorString, status)
				case dfuStatusErrorVendor:
					return fmt.Errorf("%s, attempting dnload cmd to a read protected sector, code string: %s", errorString, status)
				default:
					return fmt.Errorf("%s, unexpected status: %d, code string: %s", errorString, status.bStatus, status)
				}
			} else {
				return fmt.Errorf("%s, unexpected state: %d, code string: %s", errorString, status.bState, status)
			}
		}
	}
	return err
}

func (d DFUDevice) MultiPageErase(addr, startPage, pagesToErase, pageSize uint, progressMessage string) error {
	d.progressBars.setStatus(progressMessage)
	d.progressBars.setMax(pagesToErase)
	d.progressBars.setIncrement(1)
	d.progressBars.reset()

	for numPages := uint(0); numPages < pagesToErase; numPages++ {
		err := d.PageErase(addr + ((startPage + numPages) * pageSize))

		if err != nil {
			return err
		}

		d.progressBars.increment()
	}
	return nil
}

func (d DFUDevice) PageErase(addr uint) error {
	cmdBuffer := make([]byte, 4)

	binary.LittleEndian.PutUint32(cmdBuffer[:], uint32(addr))

	err := d.dnloadSpecialCommand(dnloadCmdErase, cmdBuffer)

	if err != nil {
		return fmt.Errorf("Page Erase Error address 0x%x: %v", addr, err)
	}
	return nil
}

func (d DFUDevice) MassErase() error {
	err := d.dnloadSpecialCommand(dnloadCmdErase, []byte{})

	if err != nil {
		return fmt.Errorf("Mass Erase Error: %v", err)
	}
	return nil
}

func (d DFUDevice) SetAddress(addr uint) error {
	cmdBuffer := make([]byte, 4)
	binary.LittleEndian.PutUint32(cmdBuffer[:], uint32(addr))
	err := d.dnloadSpecialCommand(dnloadCmdSetAddress, cmdBuffer)

	if err != nil {
		return fmt.Errorf("Set Address 0x%x Error: %v", addr, err)
	}
	return nil
}

func (d DFUDevice) WriteMemory(addr uint, data []byte, progressMessage string) error {
	err := d.SetAddress(addr)

	if err != nil {
		return fmt.Errorf("Error in SetAddress of Write Memory: %v", err)
	}

	//block size, write in max block size (2048 bytes)
	transferSize := 2048
	bytesLeftToTransfer := len(data)

	d.progressBars.setStatus(progressMessage)
	d.progressBars.setMax(uint(bytesLeftToTransfer))
	d.progressBars.setIncrement(uint(transferSize))
	d.progressBars.reset()

	//Block num starts at 2 to signal dnload() that it is a write command per spec
	blockNum := uint16(0)

	if bytesLeftToTransfer <= transferSize {
		err = d.dnloadCommand(blockNum+2, data)
		return err
	}

	//address = ((wValue - 2) * transferSize) + addr
	for bytesLeftToTransfer > 0 {
		//thisAddr := int(blockNum)*transferSize + int(addr)
		//final transfer is less than transfer size, must reset address
		if bytesLeftToTransfer < transferSize {
			dataSlice := data[transferSize*int(blockNum):]
			err := d.SetAddress(addr)

			d.progressBars.setIncrement(uint(bytesLeftToTransfer))

			if err != nil {
				return fmt.Errorf("Error in final SetAddress of Write Memory: %v", err)
			}

			//fmt.Printf("Writing to 0x%x, bytes left : %d\r\n", thisAddr, 0)
			err = d.dnloadCommand(blockNum+2, dataSlice)
			if err != nil {
				return fmt.Errorf("Write failed after final dnload address 0x%x: %v", int(blockNum)*transferSize+int(addr), err)
			}

			d.progressBars.increment()

			return err
		}
		dataSlice := data[transferSize*int(blockNum) : transferSize*(int(blockNum)+1)]
		//fmt.Printf("Writing to 0x%x, bytes left : %d\r\n", thisAddr, bytesLeftToTransfer)

		//Transfer next block
		err = d.dnloadCommand(blockNum+2, dataSlice)

		if err != nil {
			return fmt.Errorf("Write failed after dnload address 0x%x: %v", int(blockNum)*transferSize+int(addr), err)
		}
		d.progressBars.increment()
		bytesLeftToTransfer -= transferSize
		blockNum++
	}

	return err
}

//func writePage() {
//}

func (d DFUDevice) ExitDFU(addr uint) error {
	err := d.SetAddress(addr)

	if err != nil {
		return fmt.Errorf("Error in exit DFU: %v", err)
	}

	d.dnloadWaitOnIdle()

	//Transfer next block
	_, err = d.dev.Control(0x21, cmdDNLOAD, 0, dfuINTERFACE, nil)

	status, err := d.GetStatus()

	if status.bState != dfuStateDfuManifest {
		return fmt.Errorf("Failed to leave DFU mode: %v", err)
	}

	return err
}

func (d DFUDevice) uploadWaitOnIdle() error {
	var status dfuStatus
	var err error

	//TODO: Implement timeouts
	for true {
		//Check that the device is in the IDLE or DNLOAD-IDLE states before proceeding
		status, err = d.GetStatus()

		if err != nil {
			return fmt.Errorf("Initial GetStatus() failed in dnload command: %v: ", err)
		}

		if status.bState != dfuStateDfuIdle && status.bState != dfuStateDfuUploadIdle {
			//fmt.Printf("State is not set for upload, found to be: %d, string: %s\r\nAttempting to clear status...\r\n", status.bState, status)
			d.ClearStatus()
		} else {
			break
		}
	}

	return err
}

func (d DFUDevice) ReadMemory(addr, length uint, progressMessage string) ([]byte, error) {
	data := make([]byte, length)

	if length == 0 {
		return data, nil
	}

	err := d.SetAddress(addr)

	if err != nil {
		return data, fmt.Errorf("Error in Read Memory: %v", err)
	}

	//block size, write in max block size (2048 bytes)
	transferSize := 2048
	blockNum := uint16(0)
	bytesLeftToTransfer := int(length)

	d.progressBars.reset()
	d.progressBars.setStatus(progressMessage)
	d.progressBars.setMax(uint(bytesLeftToTransfer))

	//Entire buffer fits in a single
	if int(length) < transferSize {
		d.progressBars.setIncrement(uint(bytesLeftToTransfer))

		err = d.uploadWaitOnIdle()

		if err != nil {
			return data, err
		}

		_, err = d.dev.Control(0xA1, cmdUPLOAD, blockNum+2, dfuINTERFACE, data)

		d.progressBars.increment()

		return data, err
	}

	d.progressBars.setIncrement(uint(transferSize))

	//address = ((wValue - 2) * transferSize) + addr
	for bytesLeftToTransfer > 0 {

		err = d.uploadWaitOnIdle()

		if err != nil {
			return data, err
		}

		//thisAddr := int(blockNum)*transferSize + int(addr)
		//final transfer is less than transfer size, must reset address
		if bytesLeftToTransfer < transferSize {

			d.progressBars.setIncrement(uint(bytesLeftToTransfer))

			dataSlice := data[transferSize*int(blockNum):]
			err := d.SetAddress(addr)

			if err != nil {
				return nil, fmt.Errorf("Error in final SetAddress of Read Memory: %v", err)
			}

			err = d.uploadWaitOnIdle()

			if err != nil {
				return data, err
			}

			//fmt.Printf("Reading from 0x%x, bytes left : %d\r\n", thisAddr, 0)
			_, err = d.dev.Control(0xA1, cmdUPLOAD, blockNum+2, dfuINTERFACE, dataSlice)

			if err != nil {
				return nil, fmt.Errorf("Read failed after final upload address 0x%x: %v", int(blockNum)*transferSize+int(addr), err)
			}

			d.progressBars.increment()

			return data, err
		}

		dataSlice := data[transferSize*int(blockNum) : transferSize*(int(blockNum)+1)]

		//fmt.Printf("Reading from 0x%x, bytes left : %d\r\n", thisAddr, bytesLeftToTransfer)

		//Transfer next block
		_, err = d.dev.Control(0xA1, cmdUPLOAD, blockNum+2, dfuINTERFACE, dataSlice)

		d.progressBars.increment()

		if err != nil {
			return nil, fmt.Errorf("Read failed after dnload address 0x%x: %v", int(blockNum)*transferSize+int(addr), err)
		}
		bytesLeftToTransfer -= transferSize
		blockNum++
	}
	return data, err
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
