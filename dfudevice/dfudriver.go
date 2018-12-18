package dfudevice

type dfuDriver interface {
	List(VID, PID uint) []string
	Open(vid, pid uint16) (device DFUDevice, err error)
	Control(rType, request uint8, val, idx uint16, data []byte) (int, error)
	InterfaceDescription(cfgNum, intfNum, altNum int) (string, error)
	Close()
}

var dfuDriverList []dfuDriver

func addDriver(driver dfuDriver) {
	if dfuDriverList == nil {
		dfuDriverList = make([]dfuDriver, 0)
	}

	dfuDriverList = append(dfuDriverList, driver)
}
