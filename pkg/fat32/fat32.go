package fat32

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/zni/fslib/internal/utilities"
)

const backup_bpb_sector uint16 = 6

const attr_readonly uint8 = 0x01
const attr_hidden uint8 = 0x02
const attr_system uint8 = 0x04
const attr_volume_id uint8 = 0x08
const attr_directory uint8 = 0x10
const attr_archive uint8 = 0x20

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

type File struct {
	Name      string
	Content   []uint8
	_ldir_loc int64
	_dir_loc  int64
	LDIREntry []*LDIR
	DIREntry  *DIR
}

type FileSystem interface {
	ReadFile(path string) (*File, error)
	CreateDir(path string) (*File, error)
	PrintInfo()
}

type FSFile interface {
	PrintInfo()
}

type FAT32 struct {
	BPB          *BPB
	FSInfo       *FSInfo
	BackupBPB    *BPB
	BackupFSInfo *FSInfo
	FAT          []uint32
	BackupFAT    []uint32
	file         *os.File
}

func Load(path string) (*FAT32, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, errors.New("could not open volume")
	}

	bpb, err := readBPB(file)
	if err != nil {
		return nil, errors.New("failed to read BPB")
	}

	fsinfo, err := readFSInfo(file)
	if err != nil {
		return nil, errors.New("failed to read FSInfo")
	}

	_, err = file.Seek(int64(bpb.bytes_per_sector*backup_bpb_sector), io.SeekStart)
	if err != nil {
		return nil, errors.New("failed to seek to backup volume information")
	}

	backup_bpb, err := readBPB(file)
	if err != nil {
		return nil, errors.New("failed to read backup BPB")
	}
	backup_fsinfo, err := readFSInfo(file)
	if err != nil {
		return nil, errors.New("failed to read backup FSInfo")
	}

	_, err = file.Seek(int64(bpb.reserved_sector_count)*int64(bpb.bytes_per_sector), io.SeekStart)
	if err != nil {
		return nil, errors.New("failed to seek to FAT")
	}

	data_sectors := bpb.total_sectors_32 - (uint32(bpb.reserved_sector_count) + uint32(bpb.number_of_fats)*bpb.fat_size_32)
	max_clusters := (data_sectors / uint32(bpb.sectors_per_cluster)) + 1
	fat, err := readFAT(
		file,
		max_clusters,
	)
	if err != nil {
		return nil, errors.New("failed to read FAT")
	}

	backup_fat, err := readFAT(
		file,
		max_clusters,
	)
	if err != nil {
		return nil, errors.New("failed to read backup FAT")
	}

	return &FAT32{bpb, fsinfo, backup_bpb, backup_fsinfo, fat, backup_fat, file}, nil
}

func lookupClusterBytes(fs *FAT32, cluster uint) int64 {
	var reserved_bytes int64 = int64(fs.BPB.reserved_sector_count * fs.BPB.bytes_per_sector)
	var fat_bytes int64 = int64(2 * (fs.BPB.fat_size_32 * uint32(fs.BPB.bytes_per_sector)))
	data_sector := reserved_bytes + fat_bytes

	var current_cluster int64 = int64(cluster - uint(fs.BPB.root_cluster))
	var cluster_size int64 = int64(fs.BPB.bytes_per_sector * uint16(fs.BPB.sectors_per_cluster))
	cluster_sector := current_cluster * cluster_size

	return (data_sector + cluster_sector)
}

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

func getFile(fs *FAT32) (*File, error) {
	var ldirs []*LDIR

	// Get location of the first LDIR.
	ldir_loc, err := fs.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	lname_entry, err := readLDIR(fs)
	if err != nil {
		return nil, err
	}
	is_long_entry := (lname_entry.attr & long_entry) == long_entry
	var name string
	if is_long_entry {
		ldirs = append(ldirs, lname_entry)
		ldir_count := int(lname_entry.ordinal^last_long_entry) - 1
		for i := 0; i < ldir_count; i++ {
			lname_entry, err = readLDIR(fs)
			if err != nil {
				return nil, err
			}
			ldirs = append(ldirs, lname_entry)
		}

		name = joinLDIRs(ldirs)
	} else {
		fs.file.Seek(-32, io.SeekCurrent)
	}

	// Get location of the DIR entry.
	dir_loc, err := fs.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	dir_entry, err := readDIR(fs)
	if err != nil {
		return nil, err
	}

	if name == "" {
		name = strings.Trim(string(dir_entry.name), " ")
	}

	return &File{name, nil, ldir_loc, dir_loc, ldirs, dir_entry}, nil
}

/*
Read a file from the filesystem given by the path.
*/
func (fs FAT32) ReadFile(file_path string) (*File, error) {
	// Start in the root cluster and calculate the cluster boundary.
	current_cluster := uint(fs.BPB.root_cluster)
	_, err := fs.file.Seek(lookupClusterBytes(&fs, current_cluster), io.SeekStart)
	if err != nil {
		return nil, err
	}
	cluster_boundary := lookupClusterBytes(&fs, current_cluster+1)

	// Split the path on forward slashes.
	// If we only have one element '/', which becomes "", then return.
	segmented_path := strings.Split(file_path, "/")
	if segmented_path[0] == "" && len(segmented_path) == 1 {
		return nil, errors.New("no file specified")
	}

	var file *File
	segmented_path_len := len(segmented_path)
	for i, s := range segmented_path {
		// If we have a leftover from the slash split, just continue.
		if s == "" {
			continue
		}

		// Get our current position again for the for loop below.
		current_location, err := fs.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}

		// While we're not at the cluster boundary and we haven't found
		// the file yet, keep looking.
		for current_location < cluster_boundary {
			file, err = getFile(&fs)
			if err != nil {
				return nil, err
			}

			// If we have a match, break out so we can analyze it.
			if file.Name == s {
				break
			}

			current_location, err = fs.file.Seek(0, io.SeekCurrent)
			if err != nil {
				return nil, err
			}
		}

		// If the file matches part of the path and it's a directory, get ready to descend.
		if file.Name == s && ((file.DIREntry.attr & attr_directory) == attr_directory) {
			cluster := utilities.DirClusterToUint(
				uint(file.DIREntry.cluster_lo),
				uint(file.DIREntry.cluster_hi),
			)
			_, err = fs.file.Seek(lookupClusterBytes(&fs, cluster), io.SeekStart)
			if err != nil {
				return nil, err
			}
			cluster_boundary = lookupClusterBytes(&fs, cluster+1)
		}

		// If we're at the last part of the path, we found the file.
		if (file.Name == s) && ((i + 1) == segmented_path_len) {
			return file, nil
		}
	}

	return nil, errors.New("file not found")
}

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

func createWriteTime() (uint16, uint16) {
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

func createDIR(name string) (*DIR, error) {
	dir_format_name, err := createDIRName(name, false)
	if err != nil {
		return nil, err
	}

	write_time, write_date := createWriteTime()
	return &DIR{dir_format_name, attr_directory, 0, 0, 0, 0, 0, 0, write_time, write_date, 0, 0}, nil
}

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

func getNextFreeCluster(fs *FAT32) (uint32, error) {
	for i := 2; i < len(fs.FAT); i++ {
		if fs.FAT[i] == 0 {
			return uint32(i), nil
		}
	}

	return 0, errors.New("no free clusters")
}

func getNextFreeDIR(fs *FAT32, cluster int64) (int64, error) {
	current_location, err := fs.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1, err
	}

	cluster_boundary := lookupClusterBytes(fs, uint(cluster+1))
	for current_location < cluster_boundary {
		dir, err := readDIR(fs)
		if err != nil {
			return -1, nil
		}

		if dir.name[0] == 0x00 {
			return current_location, nil
		}

		current_location, err = fs.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return -1, err
		}
	}

	return -1, errors.New("no free space in cluster")
}

func writeLDIRs(fs *FAT32, ldirs []*LDIR, loc int64) (int64, error) {
	if _, err := fs.file.Seek(loc, io.SeekStart); err != nil {
		return -1, err
	}

	for _, ldir := range ldirs {
		if _, err := fs.file.Write([]byte{ldir.ordinal}); err != nil {
			return -1, err
		}
		if _, err := fs.file.Write(ldir.name1); err != nil {
			return -1, err
		}
		if _, err := fs.file.Write([]byte{ldir.attr}); err != nil {
			return -1, err
		}
		if _, err := fs.file.Write([]byte{ldir.ltype}); err != nil {
			return -1, err
		}
		if _, err := fs.file.Write([]byte{ldir.chksum}); err != nil {
			return -1, err
		}
		if _, err := fs.file.Write(ldir.name2); err != nil {
			return -1, err
		}
		if _, err := fs.file.Write(utilities.ShortToBytes(ldir.cluster_lo)); err != nil {
			return -1, err
		}
		if _, err := fs.file.Write(ldir.name3); err != nil {
			return -1, err
		}
	}

	ldir_end_loc, err := fs.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1, err
	}
	return ldir_end_loc, nil
}

func writeDIR(fs *FAT32, dir *DIR, loc int64) (int64, error) {
	if _, err := fs.file.Seek(loc, io.SeekStart); err != nil {
		return -1, err
	}

	if _, err := fs.file.Write(dir.name); err != nil {
		return -1, err
	}
	if _, err := fs.file.Write([]uint8{dir.attr}); err != nil {
		return -1, err
	}
	if _, err := fs.file.Write([]uint8{dir.ntres}); err != nil {
		return -1, err
	}
	if _, err := fs.file.Write([]uint8{dir.crt_time_tenth}); err != nil {
		return -1, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.crt_time)); err != nil {
		return -1, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.crt_date)); err != nil {
		return -1, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.lst_acc_date)); err != nil {
		return -1, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.cluster_hi)); err != nil {
		return -1, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.wrt_time)); err != nil {
		return -1, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.wrt_date)); err != nil {
		return -1, err
	}
	if _, err := fs.file.Write(utilities.ShortToBytes(dir.cluster_lo)); err != nil {
		return -1, err
	}
	if _, err := fs.file.Write(utilities.IntToBytes(dir.filesize)); err != nil {
		return -1, err
	}

	end_loc, err := fs.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1, err
	}

	return end_loc, nil
}

func markEOC(fs *FAT32, cluster uint) {
	fs.FAT[cluster] = fs.FAT[1]
}

func zeroCluster(fs *FAT32, cluster_loc int64) error {
	if _, err := fs.file.Seek(cluster_loc, io.SeekStart); err != nil {
		return err
	}

	cluster_size := fs.BPB.bytes_per_sector * uint16(fs.BPB.sectors_per_cluster)
	if _, err := fs.file.Write(make([]byte, cluster_size)); err != nil {
		return err
	}

	return nil
}

func syncFileSystemData(fs *FAT32) error {
	// Jump to FSInfo block
	if err := seekToFSInfo(fs); err != nil {
		return err
	}

	// Write out FSInfo block
	if err := writeFSInfo(fs); err != nil {
		return err
	}

	// Seek to FAT
	if err := seekToFAT(fs); err != nil {
		return err
	}

	// Write out FAT and Backup FAT
	if err := writeFAT(fs); err != nil {
		return err
	}

	return nil
}

func (fs FAT32) CreateDir(dir_path string) (*File, error) {
	// Get the base path before our new directory.
	dir_name := path.Base(dir_path)
	if (dir_name == "/") || (dir_name == ".") {
		return nil, errors.New("invalid path")
	}

	// Read in the information for the containing directory.
	base_dir, err := fs.ReadFile(path.Dir(dir_path))
	if err != nil {
		return nil, err
	}

	// Create our DIR and LDIR entries.
	dir_entry, err := createDIR(dir_name)
	if err != nil {
		return nil, err
	}
	chksum := computeShortChecksum(dir_entry)
	ldirs, err := createLDIRs(dir_name, chksum)
	if err != nil {
		return nil, err
	}

	// Compute the next free cluster and free space for our
	// DIR and LDIR entries.
	base_dir_cluster := utilities.DirClusterToUint(
		uint(base_dir.DIREntry.cluster_lo),
		uint(base_dir.DIREntry.cluster_hi),
	)
	cluster_bytes := lookupClusterBytes(&fs, base_dir_cluster)
	free_cluster, err := getNextFreeCluster(&fs)
	if err != nil {
		return nil, err
	}
	free_cluster_bytes := lookupClusterBytes(&fs, uint(free_cluster))
	next_free_bytes, err := getNextFreeDIR(&fs, cluster_bytes)
	if err != nil {
		return nil, err
	}
	dir_entry.cluster_lo = uint16(free_cluster & 0x0000FFFF)
	dir_entry.cluster_hi = uint16((free_cluster & 0xFFFF0000) >> 16)

	// Write out the LDIR and DIR entries for the directory.
	ldir_end_location, err := writeLDIRs(&fs, ldirs, next_free_bytes)
	if err != nil {
		return nil, err
	}
	_, err = writeDIR(&fs, dir_entry, ldir_end_location)
	if err != nil {
		return nil, err
	}

	// Zero the cluster where we'll store the contents of the new directory.
	if err := zeroCluster(&fs, free_cluster_bytes); err != nil {
		return nil, err
	}

	// Create and write out the '.' and '..' entries.
	dot_dir_name, _ := createDIRName(".", true)
	dot_dir := DIR{
		dot_dir_name,
		attr_directory,
		0, 0, 0, 0, 0,
		dir_entry.cluster_hi,
		dir_entry.wrt_time,
		dir_entry.wrt_date,
		dir_entry.cluster_lo,
		0,
	}
	dot_dir_end_loc, err := writeDIR(&fs, &dot_dir, free_cluster_bytes)
	if err != nil {
		return nil, err
	}
	dotdot_dir_name, _ := createDIRName("..", true)
	dotdot_dir := DIR{
		dotdot_dir_name,
		attr_directory,
		0, 0, 0, 0, 0,
		base_dir.DIREntry.cluster_hi,
		dir_entry.wrt_time,
		dir_entry.wrt_date,
		base_dir.DIREntry.cluster_lo,
		0,
	}
	if _, err = writeDIR(&fs, &dotdot_dir, dot_dir_end_loc); err != nil {
		return nil, err
	}

	// Mark the cluster with the '.' and '..' entries as end of cluster.
	markEOC(&fs, uint(free_cluster))

	// Update the FSInfo with the next free cluster and new free cluster count.
	next_free_cluster, err := getNextFreeCluster(&fs)
	if err != nil {
		return nil, err
	}
	fs.FSInfo.next_free = next_free_cluster
	fs.FSInfo.free_count = fs.FSInfo.free_count - 1

	// Write out the updated FSInfo and FATs.
	if err := syncFileSystemData(&fs); err != nil {
		return nil, err
	}

	// Return a File representation of the new directory.
	return &File{dir_name, nil, next_free_bytes, ldir_end_location, ldirs, dir_entry}, nil
}

func (fs *FAT32) Close() error {
	if err := fs.file.Close(); err != nil {
		return err
	} else {
		return nil
	}
}

func (fs FAT32) PrintInfo() {
	fmt.Printf("+---------------------+\n")
	fmt.Printf("|  VOLUME DEBUG INFO  |\n")
	fmt.Printf("+---------------------+\n")
	fmt.Printf("\\ volume_filename: %s\n", path.Base(fs.file.Name()))
	fmt.Printf("\\ bytes_per_sector: %d\n", fs.BPB.bytes_per_sector)
	fmt.Printf("\\ sectors_per_cluster: %d\n", fs.BPB.sectors_per_cluster)
	fmt.Printf("\\ volume_label: %v\n", string(fs.BPB.volume_label))
	fmt.Printf("\\ file_sys_type: %v\n", string(fs.BPB.file_sys_type))
	fmt.Printf("\\ free_clusters: %v\n", fs.FSInfo.free_count)
	fmt.Printf("\\ next_free_cluster: %v\n", fs.FSInfo.next_free)
	fmt.Println("")
}

func (file *File) PrintInfo() {
	fmt.Printf("+-------------------+\n")
	fmt.Printf("|  FILE DEBUG INFO  |\n")
	fmt.Printf("+-------------------+\n")
	fmt.Printf("\\ filename  : %s\n", file.Name)
	fmt.Printf("\\ LDIR loc  : %08x\n", file._ldir_loc)
	fmt.Printf("\\ DIR loc   : %08x\n", file._dir_loc)
	if (file.DIREntry.attr & attr_directory) == attr_directory {
		fmt.Printf("\\ directory?: true\n")
	} else {
		fmt.Printf("\\ directory?: false\n")
	}
	fmt.Printf("\\ cluster   : %d\n",
		utilities.DirClusterToUint(
			uint(file.DIREntry.cluster_lo),
			uint(file.DIREntry.cluster_hi),
		),
	)
	fmt.Println("")
}
