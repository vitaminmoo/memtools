package pidmaps

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	PermRead          = 0b1000
	PermWrite         = 0b0100
	PermExecute       = 0b0010
	PermPrivateShared = 0b0001
)

func Maps(pid int) ([]Map, error) {
	file, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var maps []Map
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}

		// Parse address range
		addrParts := strings.Split(parts[0], "-")
		if len(addrParts) != 2 {
			continue
		}

		var addressStart, addressEnd uint64
		if _, err := fmt.Sscanf(addrParts[0], "%x", &addressStart); err != nil {
			continue
		}
		if _, err := fmt.Sscanf(addrParts[1], "%x", &addressEnd); err != nil {
			continue
		}

		// Parse permissions
		var perms int8
		permStr := parts[1]
		if len(permStr) >= 4 {
			if permStr[0] == 'r' {
				perms |= PermRead
			}
			if permStr[1] == 'w' {
				perms |= PermWrite
			}
			if permStr[2] == 'x' {
				perms |= PermExecute
			}
			if permStr[3] == 's' {
				perms |= PermPrivateShared
			}
		}

		// Parse offset
		var offset uint64
		if _, err := fmt.Sscanf(parts[2], "%x", &offset); err != nil {
			continue
		}

		// Parse device major:minor
		devParts := strings.Split(parts[3], ":")
		if len(devParts) != 2 {
			continue
		}
		var devMajor, devMinor int
		if _, err := fmt.Sscanf(devParts[0], "%x", &devMajor); err != nil {
			continue
		}
		if _, err := fmt.Sscanf(devParts[1], "%x", &devMinor); err != nil {
			continue
		}

		// Parse inode
		var inode int64
		if _, err := fmt.Sscanf(parts[4], "%d", &inode); err != nil {
			continue
		}

		// Parse pathname (optional)
		var pathName string
		if len(parts) > 5 {
			pathName = strings.Join(parts[5:], " ")
		}

		maps = append(maps, Map{
			addressStart: uintptr(addressStart),
			addressEnd:   uintptr(addressEnd),
			perms:        perms,
			offset:       uintptr(offset),
			devMajor:     devMajor,
			devMinor:     devMinor,
			inode:        inode,
			pathName:     pathName,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading /proc/%d/maps: %w", pid, err)
	}
	return maps, nil
}

type Map struct {
	addressStart uintptr
	addressEnd   uintptr
	perms        int8
	offset       uintptr
	devMajor     int
	devMinor     int
	inode        int64
	pathName     string
}

func (m Map) Start() uintptr {
	return m.addressStart
}
func (m Map) End() uintptr {
	return m.addressEnd
}
func (m Map) Contains(addr uintptr) bool {
	return m.Start() <= addr && addr < m.End()
}
func (m Map) PermRead() bool {
	return m.perms&PermRead != 0
}
func (m Map) PermWrite() bool {
	return m.perms&PermWrite != 0
}

func (m Map) PermExecute() bool {
	return m.perms&PermExecute != 0
}

func (m Map) PermPrivate() bool {
	return m.perms&PermPrivateShared == 0
}

func (m Map) PermShared() bool {
	return m.perms&PermPrivateShared != 0
}

func (m Map) Offset() uintptr {
	return m.offset
}

func (m Map) DevMajor() int {
	return m.devMajor
}

func (m Map) DevMinor() int {
	return m.devMinor
}

func (m Map) Dev() string {
	return fmt.Sprintf("%02d:%02d", m.DevMajor(), m.DevMinor())
}

func (m Map) Inode() int64 {
	return m.inode
}

func (m Map) PathName() string {
	return strings.TrimSuffix(m.pathName, " (deleted)")
}

func (m Map) PathNameDeleted() bool {
	return strings.HasSuffix(m.pathName, " (deleted)")
}

func (m Map) Anonymous() bool {
	return m.pathName == ""
}
