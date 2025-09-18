package fat

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/zni/fslib/internal/utilities"
	fs "github.com/zni/fslib/pkg/fs/common"
)

const backup_bpb_sector uint16 = 6

/*
Load a volume's information into memory.
*/
func Load(path string) (*FAT32, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, &fs.FSError{Op: "Load", Path: path, Err: err}
	}

	bpb, err := ReadBPB32(file)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "Load",
			Path: path,
			Err:  fmt.Errorf("failed to read BPB: %w", err),
		}
	}

	var fsinfo FSInfo
	err = fsinfo.Read(file)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "Load",
			Path: path,
			Err:  fmt.Errorf("failed to read FSInfo: %w", err),
		}
	}

	backup_bpb_seek := int64(bpb.Common.BPB_bytspersec * backup_bpb_sector)
	_, err = file.Seek(backup_bpb_seek, io.SeekStart)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "Load",
			Path: path,
			Err:  fmt.Errorf("failed to read backup BPB: %w", err),
		}
	}

	backup_bpb, err := ReadBPB32(file)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "Load",
			Path: path,
			Err:  fmt.Errorf("failed to read backup BPB: %w", err),
		}
	}

	var backup_fsinfo FSInfo
	err = backup_fsinfo.Read(file)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "Load",
			Path: path,
			Err:  fmt.Errorf("failed to read backup FSInfo: %w", err),
		}
	}

	fat_seek := int64(bpb.Common.BPB_rsvdseccnt) * int64(bpb.Common.BPB_bytspersec)
	_, err = file.Seek(fat_seek, io.SeekStart)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "Load",
			Path: path,
			Err:  fmt.Errorf("failed to read FAT: %w", err),
		}
	}

	data_sectors := bpb.Common.BPB_totsec32 - (uint32(bpb.Common.BPB_rsvdseccnt) + uint32(bpb.Common.BPB_numfats)*bpb.Extended.BPB_fatsz32)
	max_clusters := (data_sectors / uint32(bpb.Common.BPB_secperclus)) + 1
	fat := MakeFAT32(max_clusters)
	err = fat.ReadFAT(
		file,
		max_clusters,
	)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "Load",
			Path: path,
			Err:  fmt.Errorf("failed to read FAT: %w", err),
		}
	}

	backup_fat := MakeFAT32(max_clusters)
	err = backup_fat.ReadFAT(
		file,
		max_clusters,
	)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "Load",
			Path: path,
			Err:  fmt.Errorf("failed to read backup FAT: %w", err),
		}
	}

	return &FAT32{
		BPB:          bpb,
		FSInfo:       &fsinfo,
		BackupBPB:    backup_bpb,
		BackupFSInfo: &backup_fsinfo,
		FAT:          fat,
		BackupFAT:    backup_fat,
		DiskRef:      file,
	}, nil
}

/*
Read a file from the volume given by the path.
*/
func (vol *FAT32) ReadFile(file_path string) (*FATFile, error) {
	// Start in the root cluster and calculate the cluster boundary.
	current_cluster := vol.BPB.Extended.BPB_rootclus
	_, err := vol.DiskRef.Seek(int64(LookupClusterBytes(vol, current_cluster)), io.SeekStart)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "ReadFile",
			Path: file_path,
			Err:  fmt.Errorf("failed to seek to cluster: %w", err),
		}
	}
	cluster_boundary := LookupClusterBytes(vol, current_cluster+1)

	// Split the path on forward slashes.
	// If we only have one element '/', which becomes "", then return.
	segmented_path := strings.Split(file_path, "/")
	if segmented_path[0] == "" && len(segmented_path) == 1 {
		return nil, &fs.FSError{
			Op:   "ReadFile",
			Path: file_path,
			Err:  fmt.Errorf("no file specified"),
		}
	}

	var file *FATFile
	segmented_path_len := len(segmented_path)
	for i, s := range segmented_path {
		// If we have a leftover from the slash split, just continue.
		if s == "" {
			continue
		}

		// Get our current position again for the for loop below.
		current_location, err := vol.DiskRef.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, &fs.FSError{
				Op:   "ReadFile",
				Path: file_path,
				Err:  fmt.Errorf("failed to get cursor location: %w", err),
			}
		}

		// While we're not at the cluster boundary and we haven't found
		// the file yet, keep looking.
		for current_location < int64(cluster_boundary) {
			file, err = GetFile(vol)
			if err != nil {
				return nil, &fs.FSError{
					Op:   "ReadFile",
					Path: file_path,
					Err:  fmt.Errorf("failed to get file: %w", err),
				}
			}

			// If we have a match, break out so we can analyze it.
			if file.Name == s {
				break
			}

			current_location, err = vol.DiskRef.Seek(0, io.SeekCurrent)
			if err != nil {
				return nil, &fs.FSError{
					Op:   "ReadFile",
					Path: file_path,
					Err:  fmt.Errorf("failed to get cursor location: %w", err),
				}
			}
		}

		// If the file matches part of the path and it's a directory, get ready to descend.
		if file.Name == s && IsDirectory(file.FSSpecificData.DIREntry) {
			cluster := utilities.DirClusterToUint(
				uint(file.FSSpecificData.DIREntry.DIR_cluster_lo),
				uint(file.FSSpecificData.DIREntry.DIR_cluster_hi),
			)
			_, err = vol.DiskRef.Seek(int64(LookupClusterBytes(vol, cluster)), io.SeekStart)
			if err != nil {
				return nil, &fs.FSError{
					Op:   "ReadFile",
					Path: file_path,
					Err:  fmt.Errorf("failed to seek to cluster: %w", err),
				}
			}
			cluster_boundary = LookupClusterBytes(vol, cluster+1)
		}

		// If we're at the last part of the path, we found the file.
		if (file.Name == s) && ((i + 1) == segmented_path_len) {
			return file, nil
		}
	}

	return nil, &fs.FSError{
		Op:   "ReadFile",
		Path: file_path,
		Err:  fmt.Errorf("file not found"),
	}
}

/*
Create a directory represented by the path.
*/
func (vol *FAT32) CreateDir(dir_path string) (*FATFile, error) {
	// Get the base path before our new directory.
	dir_name := path.Base(dir_path)
	if (dir_name == "/") || (dir_name == ".") {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("invalid directory path"),
		}
	}

	// Check if this filename already exists.
	if _, err := vol.ReadFile(dir_path); err == nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("file name already exists"),
		}
	}

	// Read in the information for the containing directory.
	base_dir, err := vol.ReadFile(path.Dir(dir_path))
	if err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to read parent directory: %w", err),
		}
	}

	// Create our DIR and LDIR entries.
	dir_entry, err := CreateDIR(dir_name, DIR_ATTR_DIRECTORY)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to create DIR: %w", err),
		}
	}
	chksum := computeShortChecksum(dir_entry)
	ldirs, err := CreateLDIRs(dir_name, chksum)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to create LDIRs: %w", err),
		}
	}

	// Compute the next free cluster and free space for our
	// DIR and LDIR entries.
	base_dir_cluster := utilities.DirClusterToUint(
		uint(base_dir.FSSpecificData.DIREntry.DIR_cluster_lo),
		uint(base_dir.FSSpecificData.DIREntry.DIR_cluster_hi),
	)
	cluster_bytes := LookupClusterBytes(vol, base_dir_cluster)
	free_cluster, err := vol.FAT.GetNextFreeCluster()
	if err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to get cluster: %w", err),
		}
	}
	free_cluster_bytes := LookupClusterBytes(vol, free_cluster)
	next_free_bytes, err := GetNextFreeDIR(vol, cluster_bytes)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to get next free DIR: %w", err),
		}
	}
	dir_entry.DIR_cluster_lo = uint16(free_cluster & 0x0000FFFF)
	dir_entry.DIR_cluster_hi = uint16((free_cluster & 0xFFFF0000) >> 16)

	// Write out the LDIR and DIR entries for the directory.
	ldir_end_location, err := WriteLDIRs(vol.DiskRef, ldirs, next_free_bytes)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to write LDIR entries: %w", err),
		}
	}
	_, err = WriteDIR(vol.DiskRef, dir_entry, ldir_end_location)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to write DIR entry: %w", err),
		}
	}

	// Zero the cluster where we'll store the contents of the new directory.
	if err := ZeroCluster(vol, free_cluster_bytes); err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to zero cluster: %w", err),
		}
	}

	// Create and write out the '.' and '..' entries.
	dot_dir, err := CreateSystemDIR(".")
	if err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to create '.' entry: %w", err),
		}
	}
	dot_dir_end_loc, err := WriteDIR(vol.DiskRef, dot_dir, free_cluster_bytes)
	if err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to write '.' entry: %w", err),
		}
	}

	dotdot_dir, err := CreateSystemDIR("..")
	if err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to create '..' entry: %w", err),
		}
	}
	if _, err = WriteDIR(vol.DiskRef, dotdot_dir, dot_dir_end_loc); err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to write '..' entry: %w", err),
		}
	}

	// Mark the cluster with the '.' and '..' entries as end of cluster.
	vol.FAT.MarkEOC(uint(free_cluster))

	// Update the FSInfo with the next free cluster and new free cluster count.
	next_free_cluster, err := vol.FAT.GetNextFreeCluster()
	if err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to get cluster for FSInfo: %w", err),
		}
	}
	vol.FSInfo.next_free = next_free_cluster
	vol.FSInfo.free_count = vol.FSInfo.free_count - 1

	// Write out the updated FSInfo and FATs.
	if err := SyncFileSystemData(vol); err != nil {
		return nil, &fs.FSError{
			Op:   "CreateDir",
			Path: dir_path,
			Err:  fmt.Errorf("failed to sync volume info: %w", err),
		}
	}

	// Return a File representation of the new directory.
	return &FATFile{
		Name:    dir_name,
		Content: nil,
		FSSpecificData: &FATFileData{
			LDIR_loc:  uint32(next_free_bytes),
			DIR_loc:   ldir_end_location,
			LDIREntry: ldirs,
			DIREntry:  dir_entry,
		},
	}, nil
}

/*
Close the file that represents the FAT32 volume.
*/
func (vol *FAT32) Close() error {
	if err := vol.DiskRef.Close(); err != nil {
		return &fs.FSError{
			Op:   "Close",
			Path: vol.DiskRef.Name(),
			Err:  fmt.Errorf("failed to close volume: %w", err),
		}
	} else {
		return nil
	}
}

/*
Print volume debug information.
*/
func (vol *FAT32) PrintInfo() {
	fmt.Printf("+---------------------+\n")
	fmt.Printf("|  VOLUME DEBUG INFO  |\n")
	fmt.Printf("+---------------------+\n")
	fmt.Printf("\\ volume_filename: %s\n", path.Base(vol.DiskRef.Name()))
	fmt.Printf("\\ bytes_per_sector: %d\n", vol.BPB.Common.BPB_bytspersec)
	fmt.Printf("\\ sectors_per_cluster: %d\n", vol.BPB.Common.BPB_secperclus)
	fmt.Printf("\\ volume_label: %v\n", string(vol.BPB.Extended.BS_vollab[:]))
	fmt.Printf("\\ file_sys_type: %v\n", string(vol.BPB.Extended.BS_filsystype[:]))
	fmt.Printf("\\ free_clusters: %v\n", vol.FSInfo.free_count)
	fmt.Printf("\\ next_free_cluster: %v\n", vol.FSInfo.next_free)
	fmt.Println("")
}
