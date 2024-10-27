package fat32

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strings"
	"unicode/utf16"

	"github.com/zni/fslib/internal/utilities"
)

const last_long_entry uint8 = 0x40
const long_entry uint8 = (attr_readonly | attr_hidden | attr_system | attr_volume_id)

type LDIR struct {
	ordinal    uint8
	name1      []uint8
	attr       uint8
	ltype      uint8
	chksum     uint8
	name2      []uint8
	cluster_lo uint16
	name3      []uint8
}

/*
Read a single LDIR entry from the volume.
*/
func readLDIR(fs *FAT32) (*LDIR, error) {
	var name_part LDIR
	var byte_ []uint8 = make([]uint8, 1)
	var short_ []uint8 = make([]uint8, 2)

	_, err := fs.file.Read(byte_)
	if err != nil {
		return nil, err
	}
	name_part.ordinal = byte_[0]

	name_part.name1 = make([]uint8, 10)
	_, err = fs.file.Read(name_part.name1)
	if err != nil {
		return nil, err
	}

	_, err = fs.file.Read(byte_)
	if err != nil {
		return nil, err
	}
	name_part.attr = byte_[0]

	_, err = fs.file.Read(byte_)
	if err != nil {
		return nil, err
	}
	name_part.ltype = byte_[0]

	_, err = fs.file.Read(byte_)
	if err != nil {
		return nil, err
	}
	name_part.chksum = byte_[0]

	name_part.name2 = make([]uint8, 12)
	_, err = fs.file.Read(name_part.name2)
	if err != nil {
		return nil, err
	}

	_, err = fs.file.Read(short_)
	if err != nil {
		return nil, err
	}
	name_part.cluster_lo = utilities.BytesToShort(short_)

	name_part.name3 = make([]uint8, 4)
	_, err = fs.file.Read(name_part.name3)
	if err != nil {
		return nil, err
	}

	return &name_part, nil
}

/*
Join an array of LDIRs into a string containing the filename.
*/
func joinLDIRs(ldirs []*LDIR) string {
	slices.Reverse(ldirs)
	var name []byte = make([]byte, 0, 255)
	for _, l := range ldirs {
		name0 := [][]uint8{l.name1, l.name2, l.name3}

		byte_name := bytes.Join(name0, []uint8{})
		name = append(name, byte_name...)
	}
	name = bytes.Trim(name, "\xff")
	var name_utf16 []uint16
	for i := 0; i < len(name); {
		name_hi := uint16(name[i])
		name_lo := uint16(name[i+1])
		codepoint := (name_lo << 8) | name_hi
		name_utf16 = append(name_utf16, codepoint)
		i += 2
	}

	filename := string(utf16.Decode(name_utf16))
	filename = strings.Trim(filename, "\x00")
	return filename
}

/*
Compute the checksum over the DIR entry used to validate live LDIR entries.
*/
func computeShortChecksum(dir *DIR) uint8 {
	var chksum uint8 = 0
	j := 0
	for i := 11; i != 0; i-- {
		if (chksum & 1) == 1 {
			chksum = 0x80 + (chksum >> 1) + dir.name[j]
		} else {
			chksum = 0 + (chksum >> 1) + dir.name[j]
		}
		j++
	}

	return chksum
}

/*
Given a name and checksum, generate the necessary LDIR entries to contain it.
*/
func createLDIRs(name string, chksum uint8) ([]*LDIR, error) {
	if len(name) > 255 {
		return nil, errors.New("name longer than 255 characters")
	}

	// Convert to utf16 and prep our name container.
	name_utf16 := utf16.Encode([]rune(name))
	name_utf16 = append(name_utf16, 0x0000)
	name_container := make([]uint8, 255)
	for i := 0; i < len(name_container); i += 1 {
		name_container[i] = 0xFF
	}

	// Move the utf16 name into the name container.
	j := 0
	for i := 0; i < len(name_utf16); i++ {
		name_container[j] = uint8(name_utf16[i] & 0x00FF)
		j++
		name_container[j] = uint8((name_utf16[i] & 0xFF00) >> 8)
		j++
	}

	// Compute our number of LDIR entries.
	offset := 0
	var number_of_ldirs int
	if (len(name_utf16) % 13) == 0 {
		number_of_ldirs = len(name_utf16) / 13
	} else {
		number_of_ldirs = (len(name_utf16) / 13) + 1
	}

	// Create the LDIR entries.
	var ldirs []*LDIR
	for i := 0; i < number_of_ldirs; i++ {
		var ldir LDIR

		ldir.name1 = name_container[offset : offset+10]
		ldir.name2 = name_container[offset+10 : offset+22]
		ldir.name3 = name_container[offset+22 : offset+26]

		if (i + 1) == number_of_ldirs {
			ldir.ordinal = uint8(i+1) | last_long_entry
		} else {
			ldir.ordinal = uint8(i + 1)
		}

		ldir.ltype = 0x00
		ldir.cluster_lo = 0x00
		ldir.attr = long_entry
		ldir.chksum = chksum

		ldirs = append(ldirs, &ldir)
		offset += 26
	}

	// LDIRs are stored in reverse order.
	slices.Reverse(ldirs)

	return ldirs, nil
}

/*
Write out an array of LDIRs to the location loc on disk.
*/
func writeLDIRs(fs *FAT32, ldirs []*LDIR, loc int64) (uint32, error) {
	if _, err := fs.file.Seek(loc, io.SeekStart); err != nil {
		return 0, err
	}

	for _, ldir := range ldirs {
		if _, err := fs.file.Write([]byte{ldir.ordinal}); err != nil {
			return 0, err
		}
		if _, err := fs.file.Write(ldir.name1); err != nil {
			return 0, err
		}
		if _, err := fs.file.Write([]byte{ldir.attr}); err != nil {
			return 0, err
		}
		if _, err := fs.file.Write([]byte{ldir.ltype}); err != nil {
			return 0, err
		}
		if _, err := fs.file.Write([]byte{ldir.chksum}); err != nil {
			return 0, err
		}
		if _, err := fs.file.Write(ldir.name2); err != nil {
			return 0, err
		}
		if _, err := fs.file.Write(utilities.ShortToBytes(ldir.cluster_lo)); err != nil {
			return 0, err
		}
		if _, err := fs.file.Write(ldir.name3); err != nil {
			return 0, err
		}
	}

	ldir_end_loc, err := fs.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	return uint32(ldir_end_loc), nil
}
