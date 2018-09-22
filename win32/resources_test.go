// +build windows

package win32

import "testing"

func TestReadMHz(t *testing.T) {
	mhz, err := getProcessorMHz()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("MHz = %d", mhz)
}

func TestGetSystemInfo(t *testing.T) {
	si, err := getSystemInfo()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("numprocs = %d", si.dwNumberOfProcessors)
}

func TestGlobalMemoryStatusEx(t *testing.T) {
	ms, err := globalMemoryStatusEx()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("memory used %%: %d%%", ms.dwMemoryLoad)
	t.Logf("total physical memory: %d", ms.ullTotalPhys)
	t.Logf("avail physical memory: %d", ms.ullAvailPhys)
	t.Logf("total virtual memory: %d", ms.ullTotalVirtual)
	t.Logf("avail virtual memory: %d", ms.ullAvailVirtual)
	t.Logf("total page file: %d", ms.ullTotalPageFile)
	t.Logf("avail page file: %d", ms.ullAvailPageFile)
}

func TestGetResources(t *testing.T) {
	res := GetSystemResources()
	t.Logf("CPU Count = %d", res.CPUNumCores)
	t.Logf("CPU MHz = %.2f", res.CPUMhzPercore)
	t.Logf("Total CPU MHz = %.2f", res.CPUTotalTicks)
	t.Logf("Total Physical Memory MiB = %.2f", res.MemoryTotalPhysicalKB/1024.0)
	t.Logf("Total Virtual Memory MiB = %.2f", res.MemoryTotalVirtualKB/1024.0)
}
