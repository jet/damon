// +build windows

package win32

import (
	"fmt"
	"math"
	"sync"

	"github.com/pkg/errors"
)

type SystemResources struct {
	CPUMhzPercore         float64
	CPUNumCores           int
	CPUTotalTicks         float64
	MemoryTotalPhysicalKB float64
	MemoryTotalVirtualKB  float64
}

var (
	systemResources SystemResources

	onceInit sync.Once
)

func getNumCores() (int, error) {
	si, err := getSystemInfo()
	if err != nil {
		return 0, err
	}
	return int(si.dwNumberOfProcessors), nil
}

func GetSystemResources() SystemResources {
	var err error
	onceInit.Do(func() {
		var cpuNumCores int
		cpuNumCores, err = getNumCores()
		if err != nil {
			err = fmt.Errorf("Unable to determine the number of CPU cores available: %v", err)
			return
		}
		var mhz uint32
		mhz, err = getProcessorMHz()
		if err != nil {
			err = fmt.Errorf("Unable to obtain CPU MHz: %v", err)
			return
		}
		var mem *_MEMORYSTATUSEX
		mem, err = globalMemoryStatusEx()
		if err != nil {
			err = fmt.Errorf("Unable to obtain total system memory: %v", err)
			return
		}
		systemResources = SystemResources{
			MemoryTotalPhysicalKB: float64(mem.ullTotalPhys) / float64(1024),
			MemoryTotalVirtualKB:  float64(mem.ullTotalVirtual) / float64(1024),
			CPUMhzPercore:         float64(mhz),
			CPUTotalTicks:         math.Floor(float64(cpuNumCores) * float64(mhz)),
			CPUNumCores:           cpuNumCores,
		}
	})
	if err != nil {
		panic(err)
	}
	return systemResources
}

func getProcessorMHz() (uint32, error) {
	subKey := `HARDWARE\DESCRIPTION\System\CentralProcessor\0`
	key, err := OpenRegistryKey("HKLM", subKey, RegistryKeyPermissions{Read: true})
	if err != nil {
		return 0, errors.Wrapf(err, "getProcessorMHz: could not open HKLM:%s", subKey)
	}
	defer CloseLogErr(key, fmt.Sprintf("getProcessorMHz: could not close key HKLM:%s", subKey))
	mhz, err := key.ReadDWORDValue("~MHz")
	if err != nil {
		return 0, errors.Wrapf(err, "getProcessorMHz: could not open HKLM:%s", subKey)
	}
	return mhz, nil
}
