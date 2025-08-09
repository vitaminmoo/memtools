// Package maps provides functionality to read and parse the memory mappings of a process
// from the /proc/[pid]/maps file on Linux systems.
// It defines types and methods to represent and manipulate these memory mappings.
package maps

import (
	"bufio"
	"errors"
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

var ErrInvalidAddress = errors.New("invalid address")

func Read(pid int) (Maps, error) {
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
			addressStart: addressStart,
			addressEnd:   addressEnd,
			perms:        perms,
			offset:       offset,
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
	addressStart uint64
	addressEnd   uint64
	perms        int8
	offset       uint64
	devMajor     int
	devMinor     int
	inode        int64
	pathName     string
}

func (m Map) Start() uint64 {
	return m.addressStart
}

func (m Map) End() uint64 {
	return m.addressEnd
}

func (m Map) Contains(addr uint64) bool {
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

func (m Map) Offset() uint64 {
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

type Maps []Map

func (m Maps) Start() uint64 {
	if len(m) == 0 {
		return 0
	}
	return m[0].Start()
}

func (m Maps) End() uint64 {
	if len(m) == 0 {
		return 0
	}
	return m[len(m)-1].End()
}

func (m Maps) Find(addr uint64) (Map, error) {
	for _, m := range m {
		if m.Contains(addr) {
			return m, nil
		}
	}
	return Map{}, ErrInvalidAddress
}

func (m Maps) FindNext(addr uint64) (Map, error) {
	for _, m := range m {
		if m.Start() > addr {
			return m, nil
		}
	}
	return Map{}, ErrInvalidAddress
}
