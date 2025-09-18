package fat

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/zni/fslib/internal/utilities"
)

const DIR_ATTR_READONLY uint8 = 0x01
const DIR_ATTR_HIDDEN uint8 = 0x02
const DIR_ATTR_SYSTEM uint8 = 0x04
const DIR_ATTR_VOLUME_ID uint8 = 0x08
const DIR_ATTR_DIRECTORY uint8 = 0x10
const DIR_ATTR_ARCHIVE uint8 = 0x20

type DIR struct {
	DIR_name           []uint8
	DIR_attr           uint8
	DIR_ntres          uint8
	DIR_crt_time_tenth uint8
	DIR_crt_time       uint16
	DIR_crt_date       uint16
	DIR_lst_acc_date   uint16
	DIR_cluster_hi     uint16
	DIR_wrt_time       uint16
	DIR_wrt_date       uint16
	DIR_cluster_lo     uint16
	DIR_filesize       uint32
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
func CreateDIRName(name string, system bool) ([]uint8, error) {
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
func ReadDIR(fs *os.File) (*DIR, error) {
	var byte_ []uint8 = make([]uint8, 1)
	var short_ []uint8 = make([]uint8, 2)
	var int_ []uint8 = make([]uint8, 4)

	var dir_entry DIR

	dir_entry.DIR_name = make([]uint8, 11)
	_, err := fs.Read(dir_entry.DIR_name)
	if err != nil {
		return nil, err
	}

	_, err = fs.Read(byte_)
	if err != nil {
		return nil, err
	}
	dir_entry.DIR_attr = byte_[0]

	_, err = fs.Seek(8, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = fs.Read(short_)
	if err != nil {
		return nil, err
	}
	dir_entry.DIR_cluster_hi = utilities.BytesToShort(short_)

	_, err = fs.Seek(4, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = fs.Read(short_)
	if err != nil {
		return nil, err
	}
	dir_entry.DIR_cluster_lo = utilities.BytesToShort(short_)

	_, err = fs.Read(int_)
	if err != nil {
		return nil, err
	}
	dir_entry.DIR_filesize = utilities.BytesToInt(int_)

	return &dir_entry, nil
}

/*
Generate the write time for the file being created.
*/
func CreateWriteTime() (uint16, uint16) {
	current_time := time.Now().UTC()
	seconds := 0
	minutes := current_time.Minute()
	hours := current_time.Hour()
	day := current_time.Day()
	month := int(current_time.Month())
	year := utilities.YearToFATYear(current_time.Year())

	var write_time uint16 = uint16((hours << 9) | (minutes << 5) | seconds)
	var write_date uint16 = uint16((year << 9) | (month << 5) | day)

	return write_time, write_date
}

/*
Create a DIR entry for the given name.
*/
func CreateDIR(name string, attrs uint8) (*DIR, error) {
	dir_format_name, err := CreateDIRName(name, false)
	if err != nil {
		return nil, err
	}

	write_time, write_date := CreateWriteTime()
	return &DIR{dir_format_name, attrs, 0, 0, 0, 0, 0, 0, write_time, write_date, 0, 0}, nil
}

func CreateSystemDIR(name string) (*DIR, error) {
	var attrs uint8 = (DIR_ATTR_DIRECTORY | DIR_ATTR_SYSTEM)
	dir_format_name, err := CreateDIRName(name, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create system DIR: %w", err)
	}

	write_time, write_date := CreateWriteTime()
	return &DIR{dir_format_name, attrs, 0, 0, 0, 0, 0, 0, write_time, write_date, 0, 0}, nil
}

/*
Write out a DIR entry dir to the location loc on disk.
*/
func WriteDIR(fs *os.File, dir *DIR, loc uint32) (uint32, error) {
	if _, err := fs.Seek(int64(loc), io.SeekStart); err != nil {
		return 0, err
	}

	if _, err := fs.Write(dir.DIR_name); err != nil {
		return 0, err
	}
	if _, err := fs.Write([]uint8{dir.DIR_attr}); err != nil {
		return 0, err
	}
	if _, err := fs.Write([]uint8{dir.DIR_ntres}); err != nil {
		return 0, err
	}
	if _, err := fs.Write([]uint8{dir.DIR_crt_time_tenth}); err != nil {
		return 0, err
	}
	if _, err := fs.Write(utilities.ShortToBytes(dir.DIR_crt_time)); err != nil {
		return 0, err
	}
	if _, err := fs.Write(utilities.ShortToBytes(dir.DIR_crt_date)); err != nil {
		return 0, err
	}
	if _, err := fs.Write(utilities.ShortToBytes(dir.DIR_lst_acc_date)); err != nil {
		return 0, err
	}
	if _, err := fs.Write(utilities.ShortToBytes(dir.DIR_cluster_hi)); err != nil {
		return 0, err
	}
	if _, err := fs.Write(utilities.ShortToBytes(dir.DIR_wrt_time)); err != nil {
		return 0, err
	}
	if _, err := fs.Write(utilities.ShortToBytes(dir.DIR_wrt_date)); err != nil {
		return 0, err
	}
	if _, err := fs.Write(utilities.ShortToBytes(dir.DIR_cluster_lo)); err != nil {
		return 0, err
	}
	if _, err := fs.Write(utilities.IntToBytes(dir.DIR_filesize)); err != nil {
		return 0, err
	}

	end_loc, err := fs.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}

	return uint32(end_loc), nil
}

func IsDirectory(d *DIR) bool {
	return (d.DIR_attr & DIR_ATTR_DIRECTORY) == DIR_ATTR_DIRECTORY
}

/*
Get the location for the next free DIR entry in a cluster.
*/
func GetNextFreeDIR[T FATSystem](fs T, cluster uint32) (int64, error) {
	disk_ref := fs.GetDiskRef()
	current_location, err := disk_ref.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1, err
	}

	cluster_boundary := LookupClusterBytes(fs, (cluster + 1))
	for current_location < int64(cluster_boundary) {
		dir, err := ReadDIR(disk_ref)
		if err != nil {
			return -1, nil
		}

		if dir.DIR_name[0] == 0x00 {
			return current_location, nil
		}

		current_location, err = disk_ref.Seek(0, io.SeekCurrent)
		if err != nil {
			return -1, err
		}
	}

	return -1, errors.New("no free space in cluster")
}
