package fat32

import (
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/zni/fslib/internal/utilities"
)

const attr_readonly uint8 = 0x01
const attr_hidden uint8 = 0x02
const attr_system uint8 = 0x04
const attr_volume_id uint8 = 0x08
const attr_directory uint8 = 0x10
const attr_archive uint8 = 0x20

type DIR struct {
	name           []uint8
	attr           uint8
	ntres          uint8
	crt_time_tenth uint8
	crt_time       uint16
	crt_date       uint16
	lst_acc_date   uint16
	cluster_hi     uint16
	wrt_time       uint16
	wrt_date       uint16
	cluster_lo     uint16
	filesize       uint32
}

/*
Does the rune c satisfy what FAT32 considers a valid character in a file name?
*/
func validCharacter(c rune) bool {
	var forbidden_characters []rune = []rune{
		0x22, 0x2A, 0x2B, 0x2C, 0x2E, 0x2F, 0x3A,
		0x3B, 0x3C, 0x3D, 0x3E, 0x3F, 0x5B, 0x5C,
		0x5D, 0x7C,
	}

	if c < 0x20 {
		return false
	}

	if slices.Index(forbidden_characters, c) != -1 {
		return false
	}

	return true
}

/*
Create the truncated DOS-style name for a given file.
*/
func createDIRName(name string, system bool) ([]uint8, error) {
	uppercase_name := strings.ToUpper(name)
	uppercase_name = strings.ReplaceAll(uppercase_name, " ", "")
	if len(uppercase_name) > 11 {
		uppercase_name = uppercase_name[:11]
	}

	dir_format_name := make([]uint8, 11)
	for i := 0; i < len(dir_format_name); i++ {
		dir_format_name[i] = 0x20
	}
	for i, s := range uppercase_name {
		if system {
			dir_format_name[i] = uint8(s)
		} else {
			if validCharacter(s) {
				dir_format_name[i] = uint8(s)
			} else {
				return nil, errors.New("name contains invalid characters")
			}
		}
	}

	return dir_format_name, nil
}

/*
Read a single DIR entry from the volume.
*/
func readDIR(fs *FAT32) (*DIR, error) {
	var byte_ []uint8 = make([]uint8, 1)
	var short_ []uint8 = make([]uint8, 2)
	var int_ []uint8 = make([]uint8, 4)

	var dir_entry DIR

	dir_entry.name = make([]uint8, 11)
	_, err := fs.file.Read(dir_entry.name)
	if err != nil {
		return nil, err
	}

	_, err = fs.file.Read(byte_)
	if err != nil {
		return nil, err
	}
	dir_entry.attr = byte_[0]

	_, err = fs.file.Seek(8, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = fs.file.Read(short_)
	if err != nil {
		return nil, err
	}
	dir_entry.cluster_hi = utilities.BytesToShort(short_)

	_, err = fs.file.Seek(4, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = fs.file.Read(short_)
	if err != nil {
		return nil, err
	}
	dir_entry.cluster_lo = utilities.BytesToShort(short_)

	_, err = fs.file.Read(int_)
	if err != nil {
		return nil, err
	}
	dir_entry.filesize = utilities.BytesToInt(int_)

	return &dir_entry, nil
}

/*
Create a DIR entry for the given name.
*/
func createDIR(name string) (*DIR, error) {
	dir_format_name, err := createDIRName(name, false)
	if err != nil {
		return nil, err
	}

	write_time, write_date := createWriteTime()
	return &DIR{dir_format_name, attr_directory, 0, 0, 0, 0, 0, 0, write_time, write_date, 0, 0}, nil
}

/*
Write out a DIR entry dir to the location loc on disk.
*/
func writeDIR(fs *FAT32, dir *DIR, loc uint32) (uint32, error) {
	if _, err := fs.file.Seek(int64(loc), io.SeekStart); err != nil {
		return 0, err
	}

	if _, err := fs.file.Write(dir.name); err != nil {
		return 0, err
	}
	if _, err := fs.file.Write([]uint8{dir.attr}); err != nil {
		return 0, err
	}
	if _, err := fs.file.Write([]uint8{dir.ntres}); err != nil {
		return 0, err
	}
	if _, err := fs.file.Write([]uint8{dir.crt_time_tenth}); err != nil {
		return 0, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.crt_time)); err != nil {
		return 0, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.crt_date)); err != nil {
		return 0, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.lst_acc_date)); err != nil {
		return 0, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.cluster_hi)); err != nil {
		return 0, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.wrt_time)); err != nil {
		return 0, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.wrt_date)); err != nil {
		return 0, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.cluster_lo)); err != nil {
		return 0, err
	}
	if _, err := fs.file.Write(utilities.IntToBytes(dir.filesize)); err != nil {
		return 0, err
	}

	end_loc, err := fs.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}

	return uint32(end_loc), nil
}
