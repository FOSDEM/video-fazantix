package utils

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DeviceType int

const (
	V4L2Device DeviceType = iota
	MediaDevice
)

type USBDevice struct {
	Type DeviceType
	Path string
}
type USBDeviceCollection struct {
	Devices []*USBDevice
}

func (c USBDeviceCollection) GetFirst(devType DeviceType) *USBDevice {
	sort.Slice(c.Devices, func(i, j int) bool {
		return c.Devices[i].Path < c.Devices[j].Path
	})
	for _, dev := range c.Devices {
		if dev.Type == devType {
			return dev
		}
	}
	return nil
}

func LocateUSBDevice(port string) (*USBDeviceCollection, error) {
	dir := filepath.Join("/sys/bus/usb/devices", port)
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := &USBDeviceCollection{make([]*USBDevice, 0)}

	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		// Skip everything that's not an interface
		if !strings.Contains(item.Name(), ":") {
			continue
		}

		drivers, err := os.ReadDir(filepath.Join(dir, item.Name()))
		if err != nil {
			continue
		}
		for _, driver := range drivers {
			if driver.Name() == "video4linux" {
				subdirs, err := os.ReadDir(filepath.Join(dir, item.Name(), "video4linux"))
				if err != nil {
					continue
				}
				for _, subdir := range subdirs {
					if strings.HasPrefix(subdir.Name(), "video") {
						result.Devices = append(result.Devices, &USBDevice{
							Type: V4L2Device,
							Path: filepath.Join("/dev", subdir.Name()),
						})
					}
				}
			}
			if strings.HasPrefix(driver.Name(), "media") {
				result.Devices = append(result.Devices, &USBDevice{
					Type: MediaDevice,
					Path: filepath.Join("/dev", driver.Name()),
				})
			}
		}
	}
	return result, nil
}
