package fat

import (
	"fmt"
	"io"

	"github.com/zni/fslib/internal/utilities"
	"github.com/zni/fslib/pkg/fs/common"
)

type FATFileData struct {
	LDIR_loc  uint32
	DIR_loc   uint32
	LDIREntry []*LDIR
	DIREntry  *DIR
}

type FATFile common.File[FATFileData]

/*
Print file debug information.
*/
func (file *FATFile) PrintInfo() {
	fmt.Printf("+-------------------+\n")
	fmt.Printf("|  FILE DEBUG INFO  |\n")
	fmt.Printf("+-------------------+\n")
	fmt.Printf("\\ filename  : %s\n", file.Name)
	fmt.Printf("\\ LDIR loc  : %08x\n", file.FSSpecificData.LDIR_loc)
	fmt.Printf("\\ DIR loc   : %08x\n", file.FSSpecificData.DIR_loc)
	if IsDirectory(file.FSSpecificData.DIREntry) {
		fmt.Printf("\\ directory?: true\n")
	} else {
		fmt.Printf("\\ directory?: false\n")
	}
	fmt.Printf("\\ cluster   : %d\n",
		utilities.DirClusterToUint(
			uint(file.FSSpecificData.DIREntry.DIR_cluster_lo),
			uint(file.FSSpecificData.DIREntry.DIR_cluster_hi),
		),
	)
	fmt.Printf("\\ file size : %d\n", file.FSSpecificData.DIREntry.DIR_filesize)
	fmt.Println("")
}

/*
Read a file's complete contents from the volume into the File.Content struct member.
*/
func ReadAll[T FATSystem](file *FATFile, fs T) (bytes_read int, err error) {
	file.Content = make([]uint8, file.FSSpecificData.DIREntry.DIR_filesize)
	return Read(file.Content, fs, file)
}

/*
Read a portion of a file's contents from the volume into the buffer provided.
*/
func Read[T FATSystem](b []byte, fs T, file *FATFile) (n int, err error) {
	if IsDirectory(file.FSSpecificData.DIREntry) {
		return 0, &common.FileError{
			Op:   "Read",
			Path: file.Name,
			Err:  fmt.Errorf("file must not be a directory"),
		}
	}

	common_bpb := fs.GetCommonBPB()
	var file_size int = int(file.FSSpecificData.DIREntry.DIR_filesize)
	var bytes_per_sector int = int(common_bpb.BPB_bytspersec)
	var sectors_per_cluster int = int(common_bpb.BPB_secperclus)
	var cluster_size int = bytes_per_sector * sectors_per_cluster
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
		uint(file.FSSpecificData.DIREntry.DIR_cluster_lo),
		uint(file.FSSpecificData.DIREntry.DIR_cluster_hi),
	)
	file_loc_bytes := LookupClusterBytes(fs, file_cluster)

	// Seek to first cluster for the file.
	disk_ref := fs.GetDiskRef()
	if _, err = disk_ref.Seek(int64(file_loc_bytes), io.SeekStart); err != nil {
		return 0, &common.FileError{
			Op:   "Read",
			Path: file.Name,
			Err:  fmt.Errorf("failed to seek to file cluster: %w", err),
		}
	}

	var EOC uint32
	fat_short := fs.GetFATShort()
	fat_int := fs.GetFATInt()
	if fat_short == nil {
		EOC = fat_int.GetEOC()
	} else {
		EOC = uint32(fat_short.GetEOC())
	}

	var total_bytes_read int = 0
	var next_cluster uint32 = file_cluster
	var bytes_read int = 0

	for bytes_to_read > 0 {
		bytes_read, err = disk_ref.Read(b[total_bytes_read:(total_bytes_read + read_size)])
		if err != nil {
			total_bytes_read += bytes_read
			return total_bytes_read, &common.FileError{
				Op:   "Read",
				Path: file.Name,
				Err:  fmt.Errorf("failed to read contents after %d bytes: %w", total_bytes_read, err),
			}
		} else {
			total_bytes_read += bytes_read
		}

		// We hit EOF, bail out.
		if bytes_read == 0 {
			return total_bytes_read, &common.FileError{
				Op:   "Read",
				Path: file.Name,
				Err:  fmt.Errorf("encountered unexpected end of file"),
			}
		}

		// Is the next cluster the end of chain?
		// If not, calculate the next cluster in the chain.
		if fat_short == nil {
			next_cluster = fat_int.GetCluster(uint(next_cluster))
		} else {
			next_cluster = uint32(fat_short.GetCluster(uint(next_cluster)))
		}
		if next_cluster != EOC {
			file_loc_bytes = LookupClusterBytes(fs, next_cluster)
			if _, err = disk_ref.Seek(int64(file_loc_bytes), io.SeekStart); err != nil {
				return 0, &common.FileError{
					Op:   "Read",
					Path: file.Name,
					Err:  fmt.Errorf("failed to seek to next cluster: %w", err),
				}
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
