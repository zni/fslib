package fat32

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/zni/fslib/internal/utilities"
)

const backup_bpb_sector uint16 = 6

type File struct {
	Name      string
	Content   []uint8
	_ldir_loc uint32
	_dir_loc  uint32
	LDIREntry []*LDIR
	DIREntry  *DIR
	fat32     *FAT32
}

type FileSystem interface {
	ReadFile(path string) (*File, error)
	CreateFile(path string, b []byte) (*File, error)
	CreateDir(path string) (*File, error)
	PrintInfo()
}

type FSFile interface {
	Read(b []byte)
	ReadAll()
	PrintInfo()
}

type FAT32 struct {
	BPB          *BPB32
	FSInfo       *FSInfo
	BackupBPB    *BPB32
	BackupFSInfo *FSInfo
	FAT          []uint32
	BackupFAT    []uint32
	file         *os.File
}

/*
Load a volume's information into memory.
*/
func Load(path string) (*FAT32, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, &FSError{"Load", path, err}
	}

	bpb, err := readBPB(file)
	if err != nil {
		return nil, &FSError{"Load", path, fmt.Errorf("failed to read BPB: %w", err)}
	}

	fsinfo, err := readFSInfo(file)
	if err != nil {
		return nil, &FSError{"Load", path, fmt.Errorf("failed to read FSInfo: %w", err)}
	}

	_, err = file.Seek(int64(bpb.Common.BPB_bytspersec*backup_bpb_sector), io.SeekStart)
	if err != nil {
		return nil, &FSError{"Load", path, fmt.Errorf("failed to read backup BPB: %w", err)}
	}

	backup_bpb, err := readBPB(file)
	if err != nil {
		return nil, &FSError{"Load", path, fmt.Errorf("failed to read backup BPB: %w", err)}
	}
	backup_fsinfo, err := readFSInfo(file)
	if err != nil {
		return nil, &FSError{"Load", path, fmt.Errorf("failed to read backup FSInfo: %w", err)}
	}

	_, err = file.Seek(int64(bpb.Common.BPB_rsvdseccnt)*int64(bpb.Common.BPB_bytspersec), io.SeekStart)
	if err != nil {
		return nil, &FSError{"Load", path, fmt.Errorf("failed to read FAT: %w", err)}
	}

	data_sectors := bpb.Common.BPB_totsec32 - (uint32(bpb.Common.BPB_rsvdseccnt) + uint32(bpb.Common.BPB_numfats)*bpb.Extended.BPB_fatsz32)
	max_clusters := (data_sectors / uint32(bpb.Common.BPB_secperclus)) + 1
	fat, err := readFAT(
		file,
		max_clusters,
	)
	if err != nil {
		return nil, &FSError{"Load", path, fmt.Errorf("failed to read FAT: %w", err)}
	}

	backup_fat, err := readFAT(
		file,
		max_clusters,
	)
	if err != nil {
		return nil, &FSError{"Load", path, fmt.Errorf("failed to read backup FAT: %w", err)}
	}

	return &FAT32{bpb, fsinfo, backup_bpb, backup_fsinfo, fat, backup_fat, file}, nil
}

/*
Look up the location in bytes of the given cluster.
*/
func lookupClusterBytes(fs *FAT32, cluster uint32) uint32 {
	var reserved_bytes int64 = int64(fs.BPB.Common.BPB_rsvdseccnt * fs.BPB.Common.BPB_bytspersec)
	var fat_bytes int64 = int64(2 * (fs.BPB.Extended.BPB_fatsz32 * uint32(fs.BPB.Common.BPB_bytspersec)))
	data_sector := reserved_bytes + fat_bytes

	var current_cluster int64 = int64(cluster - fs.BPB.Extended.BPB_rootclus)
	var cluster_size int64 = int64(fs.BPB.Common.BPB_bytspersec * uint16(fs.BPB.Common.BPB_secperclus))
	cluster_sector := current_cluster * cluster_size

	return uint32(data_sector + cluster_sector)
}

/*
Read a file's complete LDIR and DIR entries from the volume.
*/
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

	return &File{name, nil, uint32(ldir_loc), uint32(dir_loc), ldirs, dir_entry, fs}, nil
}

/*
Read a file from the volume given by the path.
*/
func (fs *FAT32) ReadFile(file_path string) (*File, error) {
	// Start in the root cluster and calculate the cluster boundary.
	current_cluster := fs.BPB.Extended.BPB_rootclus
	_, err := fs.file.Seek(int64(lookupClusterBytes(fs, current_cluster)), io.SeekStart)
	if err != nil {
		return nil, &FSError{"ReadFile", file_path, fmt.Errorf("failed to seek to cluster: %w", err)}
	}
	cluster_boundary := lookupClusterBytes(fs, current_cluster+1)

	// Split the path on forward slashes.
	// If we only have one element '/', which becomes "", then return.
	segmented_path := strings.Split(file_path, "/")
	if segmented_path[0] == "" && len(segmented_path) == 1 {
		return nil, &FSError{"ReadFile", file_path, fmt.Errorf("no file specified")}
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
			return nil, &FSError{"ReadFile", file_path, fmt.Errorf("failed to get cursor location: %w", err)}
		}

		// While we're not at the cluster boundary and we haven't found
		// the file yet, keep looking.
		for current_location < int64(cluster_boundary) {
			file, err = getFile(fs)
			if err != nil {
				return nil, &FSError{"ReadFile", file_path, fmt.Errorf("failed to get file: %w", err)}
			}

			// If we have a match, break out so we can analyze it.
			if file.Name == s {
				break
			}

			current_location, err = fs.file.Seek(0, io.SeekCurrent)
			if err != nil {
				return nil, &FSError{"ReadFile", file_path, fmt.Errorf("failed to get cursor location: %w", err)}
			}
		}

		// If the file matches part of the path and it's a directory, get ready to descend.
		if file.Name == s && ((file.DIREntry.attr & attr_directory) == attr_directory) {
			cluster := utilities.DirClusterToUint(
				uint(file.DIREntry.cluster_lo),
				uint(file.DIREntry.cluster_hi),
			)
			_, err = fs.file.Seek(int64(lookupClusterBytes(fs, cluster)), io.SeekStart)
			if err != nil {
				return nil, &FSError{"ReadFile", file_path, fmt.Errorf("failed to seek to cluster: %w", err)}
			}
			cluster_boundary = lookupClusterBytes(fs, cluster+1)
		}

		// If we're at the last part of the path, we found the file.
		if (file.Name == s) && ((i + 1) == segmented_path_len) {
			return file, nil
		}
	}

	return nil, &FSError{"ReadFile", file_path, fmt.Errorf("file not found")}
}

/*
Get the next free cluster from the FAT not marked EOC.
*/
func getNextFreeCluster(fs *FAT32) (uint32, error) {
	for i := 2; i < len(fs.FAT); i++ {
		if fs.FAT[i] == 0 {
			return uint32(i), nil
		}
	}

	return 0, errors.New("no free clusters")
}

/*
Get the location for the next free DIR entry in a cluster.
*/
func getNextFreeDIR(fs *FAT32, cluster uint32) (int64, error) {
	current_location, err := fs.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1, err
	}

	cluster_boundary := lookupClusterBytes(fs, (cluster + 1))
	for current_location < int64(cluster_boundary) {
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

/*
Mark a cluster in the FAT with the EOC value.
*/
func markEOC(fs *FAT32, cluster uint) {
	fs.FAT[cluster] = fs.FAT[1]
}

/*
Zero out a cluster for use.
*/
func zeroCluster(fs *FAT32, cluster uint32) error {
	if _, err := fs.file.Seek(int64(cluster), io.SeekStart); err != nil {
		return err
	}

	cluster_size := fs.BPB.Common.BPB_bytspersec * uint16(fs.BPB.Common.BPB_secperclus)
	if _, err := fs.file.Write(make([]byte, cluster_size)); err != nil {
		return err
	}

	return nil
}

/*
Write back out the FSInfo and FAT after a write operation.
*/
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

/*
Create a directory represented by the path.
*/
func (fs *FAT32) CreateDir(dir_path string) (*File, error) {
	// Get the base path before our new directory.
	dir_name := path.Base(dir_path)
	if (dir_name == "/") || (dir_name == ".") {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("invalid directory path")}
	}

	// Check if this filename already exists.
	if _, err := fs.ReadFile(dir_path); err == nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("file name already exists")}
	}

	// Read in the information for the containing directory.
	base_dir, err := fs.ReadFile(path.Dir(dir_path))
	if err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to read parent directory: %w", err)}
	}

	// Create our DIR and LDIR entries.
	dir_entry, err := createDIR(dir_name, attr_directory)
	if err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to create DIR: %w", err)}
	}
	chksum := computeShortChecksum(dir_entry)
	ldirs, err := createLDIRs(dir_name, chksum)
	if err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to create LDIRs: %w", err)}
	}

	// Compute the next free cluster and free space for our
	// DIR and LDIR entries.
	base_dir_cluster := utilities.DirClusterToUint(
		uint(base_dir.DIREntry.cluster_lo),
		uint(base_dir.DIREntry.cluster_hi),
	)
	cluster_bytes := lookupClusterBytes(fs, base_dir_cluster)
	free_cluster, err := getNextFreeCluster(fs)
	if err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to get cluster: %w", err)}
	}
	free_cluster_bytes := lookupClusterBytes(fs, free_cluster)
	next_free_bytes, err := getNextFreeDIR(fs, cluster_bytes)
	if err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to get next free DIR: %w", err)}
	}
	dir_entry.cluster_lo = uint16(free_cluster & 0x0000FFFF)
	dir_entry.cluster_hi = uint16((free_cluster & 0xFFFF0000) >> 16)

	// Write out the LDIR and DIR entries for the directory.
	ldir_end_location, err := writeLDIRs(fs, ldirs, next_free_bytes)
	if err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to write LDIR entries: %w", err)}
	}
	_, err = writeDIR(fs, dir_entry, ldir_end_location)
	if err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to write DIR entry: %w", err)}
	}

	// Zero the cluster where we'll store the contents of the new directory.
	if err := zeroCluster(fs, free_cluster_bytes); err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to zero cluster: %w", err)}
	}

	// Create and write out the '.' and '..' entries.
	dot_dir, err := createSystemDIR(".", attr_directory|attr_system)
	if err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to create '.' entry: %w", err)}
	}
	dot_dir_end_loc, err := writeDIR(fs, dot_dir, free_cluster_bytes)
	if err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to write '.' entry: %w", err)}
	}

	dotdot_dir, err := createSystemDIR("..", attr_directory|attr_system)
	if err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to create '..' entry: %w", err)}
	}
	if _, err = writeDIR(fs, dotdot_dir, dot_dir_end_loc); err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to write '..' entry: %w", err)}
	}

	// Mark the cluster with the '.' and '..' entries as end of cluster.
	markEOC(fs, uint(free_cluster))

	// Update the FSInfo with the next free cluster and new free cluster count.
	next_free_cluster, err := getNextFreeCluster(fs)
	if err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to get cluster for FSInfo: %w", err)}
	}
	fs.FSInfo.next_free = next_free_cluster
	fs.FSInfo.free_count = fs.FSInfo.free_count - 1

	// Write out the updated FSInfo and FATs.
	if err := syncFileSystemData(fs); err != nil {
		return nil, &FSError{"CreateDir", dir_path, fmt.Errorf("failed to sync volume info: %w", err)}
	}

	// Return a File representation of the new directory.
	return &File{dir_name, nil, uint32(next_free_bytes), ldir_end_location, ldirs, dir_entry, fs}, nil
}

/*
Close the file that represents the FAT32 volume.
*/
func (fs *FAT32) Close() error {
	if err := fs.file.Close(); err != nil {
		return &FSError{"Close", fs.file.Name(), fmt.Errorf("failed to close volume: %w", err)}
	} else {
		return nil
	}
}

/*
Print volume debug information.
*/
func (fs *FAT32) PrintInfo() {
	fmt.Printf("+---------------------+\n")
	fmt.Printf("|  VOLUME DEBUG INFO  |\n")
	fmt.Printf("+---------------------+\n")
	fmt.Printf("\\ volume_filename: %s\n", path.Base(fs.file.Name()))
	fmt.Printf("\\ bytes_per_sector: %d\n", fs.BPB.Common.BPB_bytspersec)
	fmt.Printf("\\ sectors_per_cluster: %d\n", fs.BPB.Common.BPB_secperclus)
	fmt.Printf("\\ volume_label: %v\n", string(fs.BPB.Extended.bs_vollab[:]))
	fmt.Printf("\\ file_sys_type: %v\n", string(fs.BPB.Extended.bs_filsystype[:]))
	fmt.Printf("\\ free_clusters: %v\n", fs.FSInfo.free_count)
	fmt.Printf("\\ next_free_cluster: %v\n", fs.FSInfo.next_free)
	fmt.Println("")
}

/*
Print file debug information.
*/
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
	fmt.Printf("\\ file size : %d\n", file.DIREntry.filesize)
	fmt.Println("")
}

/*
Read a file's complete contents from the volume into the File.Content struct member.
*/
func (file *File) ReadAll() (bytes_read int, err error) {
	file.Content = make([]uint8, file.DIREntry.filesize)
	return file.Read(file.Content)
}

/*
Read a portion of a file's contents from the volume into the buffer provided.
*/
func (file *File) Read(b []byte) (n int, err error) {
	if (file.DIREntry.attr & attr_directory) == attr_directory {
		return 0, &FileError{"Read", file.Name, fmt.Errorf("file must not be a directory")}
	}

	var file_size int = int(file.DIREntry.filesize)
	var cluster_size int = int(file.fat32.BPB.Common.BPB_bytspersec) * int(file.fat32.BPB.Common.BPB_secperclus)
	var bytes_to_read int = len(b)
	if bytes_to_read > file_size {
		bytes_to_read = file_size
	}

	var read_size int
	if bytes_to_read > cluster_size {
		read_size = cluster_size
	} else {
		read_size = bytes_to_read
	}

	file_cluster := utilities.DirClusterToUint(
		uint(file.DIREntry.cluster_lo),
		uint(file.DIREntry.cluster_hi),
	)
	file_loc_bytes := lookupClusterBytes(file.fat32, file_cluster)

	// Seek to first cluster for the file.
	if _, err = file.fat32.file.Seek(int64(file_loc_bytes), io.SeekStart); err != nil {
		return 0, &FileError{"Read", file.Name, fmt.Errorf("failed to seek to file cluster: %w", err)}
	}

	var total_bytes_read int = 0
	var EOC uint32 = file.fat32.FAT[1]
	var next_cluster uint32 = file_cluster
	var bytes_read int = 0

	for bytes_to_read > 0 {
		bytes_read, err = file.fat32.file.Read(b[total_bytes_read:(total_bytes_read + read_size)])
		if err != nil {
			total_bytes_read += bytes_read
			return total_bytes_read, &FileError{"Read", file.Name, fmt.Errorf("failed to read contents after %d bytes: %w", total_bytes_read, err)}
		} else {
			total_bytes_read += bytes_read
		}

		// We hit EOF, bail out.
		if bytes_read == 0 {
			return total_bytes_read, &FileError{"Read", file.Name, fmt.Errorf("encountered unexpected end of file")}
		}

		// Is the next cluster the end of chain?
		// If not, calculate the next cluster in the chain.
		next_cluster = file.fat32.FAT[next_cluster]
		if next_cluster != EOC {
			file_loc_bytes = lookupClusterBytes(file.fat32, next_cluster)
			if _, err = file.fat32.file.Seek(int64(file_loc_bytes), io.SeekStart); err != nil {
				return 0, &FileError{"Read", file.Name, fmt.Errorf("failed to seek to next cluster: %w", err)}
			}
		}

		bytes_to_read -= bytes_read

		// Is the file size larger than a cluster?
		// If so, read_size is cluster sized.
		// Otherwise, read_size is the remaining file size.
		if bytes_to_read > cluster_size {
			read_size = cluster_size
		} else {
			read_size = bytes_to_read
		}

	}

	return total_bytes_read, nil
}
