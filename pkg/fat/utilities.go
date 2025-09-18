package fat

import (
	"io"
	"strings"
)

/*
Look up the location in bytes of the given cluster.
*/
func LookupClusterBytes[T FATSystem](fs T, cluster uint32) uint32 {
	common_bpb := fs.GetCommonBPB()
	extended_bpb := fs.GetExtendedBPBFull()

	// Calculate the bytes taken up by the file system information.
	var reserved_bytes int64 = int64(common_bpb.BPB_rsvdseccnt * common_bpb.BPB_bytspersec)
	var fat_bytes int64
	if extended_bpb == nil {
		fat_bytes = int64(2 * (uint32(common_bpb.BPB_fatsz16) * uint32(common_bpb.BPB_bytspersec)))
	} else {
		fat_bytes = int64(2 * (extended_bpb.BPB_fatsz32 * uint32(common_bpb.BPB_bytspersec)))
	}
	data_sector := reserved_bytes + fat_bytes

	// Calculate the amount of bytes a sector takes up, multiplied by its location.
	var current_cluster int64
	if extended_bpb == nil {
		current_cluster = int64(cluster)
	} else {
		current_cluster = int64(cluster - extended_bpb.BPB_rootclus)
	}
	var cluster_size int64 = int64(common_bpb.BPB_bytspersec * uint16(common_bpb.BPB_secperclus))
	cluster_sector := current_cluster * cluster_size

	// Return the reserved bytes in addition to the bytes to the cluster.
	return uint32(data_sector + cluster_sector)
}

/*
Read a file's complete LDIR and DIR entries from the volume.
*/
func GetFile[T FATSystem](fs T) (*FATFile, error) {
	var ldirs []*LDIR
	disk_ref := fs.GetDiskRef()

	// Get location of the first LDIR.
	ldir_loc, err := disk_ref.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	lname_entry, err := ReadLDIR(disk_ref)
	if err != nil {
		return nil, err
	}
	is_long_entry := (lname_entry.attr & long_entry) == long_entry
	var name string
	if is_long_entry {
		ldirs = append(ldirs, lname_entry)
		ldir_count := int(lname_entry.ordinal^last_long_entry) - 1
		for i := 0; i < ldir_count; i++ {
			lname_entry, err = ReadLDIR(disk_ref)
			if err != nil {
				return nil, err
			}
			ldirs = append(ldirs, lname_entry)
		}

		name = joinLDIRs(ldirs)
	} else {
		disk_ref.Seek(-32, io.SeekCurrent)
	}

	// Get location of the DIR entry.
	dir_loc, err := disk_ref.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	dir_entry, err := ReadDIR(disk_ref)
	if err != nil {
		return nil, err
	}

	if name == "" {
		name = strings.Trim(string(dir_entry.DIR_name), " ")
	}

	fat_file_data := FATFileData{uint32(ldir_loc), uint32(dir_loc), ldirs, dir_entry}
	fs_file := FATFile{name, nil, &fat_file_data}

	return &fs_file, nil
}

/*
Zero out a cluster for use.
*/
func ZeroCluster[T FATSystem](fs T, cluster uint32) error {
	disk_ref := fs.GetDiskRef()
	if _, err := disk_ref.Seek(int64(cluster), io.SeekStart); err != nil {
		return err
	}

	common_bpb := fs.GetCommonBPB()
	cluster_size := common_bpb.BPB_bytspersec * uint16(common_bpb.BPB_secperclus)
	if _, err := disk_ref.Write(make([]byte, cluster_size)); err != nil {
		return err
	}

	return nil
}

/*
Write back out the FSInfo and FAT after a write operation.
*/
func SyncFileSystemData[T FATSystem](fs T) error {
	disk_ref := fs.GetDiskRef()
	common_bpb := fs.GetCommonBPB()

	// Jump to FSInfo block
	if err := seekToFSInfo(disk_ref, common_bpb); err != nil {
		return err
	}

	// Write out FSInfo block
	fsinfo := fs.GetFSInfo()
	if fsinfo != nil {
		if err := fsinfo.Write(disk_ref, common_bpb); err != nil {
			return err
		}
	}

	// Seek to FAT
	if err := SeekToFAT(disk_ref, common_bpb); err != nil {
		return err
	}

	// Write out FAT and Backup FAT
	fat_short := fs.GetFATShort()
	if fat_short != nil {
		if err := fat_short.WriteFAT(disk_ref); err != nil {
			return err
		}
		if err := fat_short.WriteFAT(disk_ref); err != nil {
			return err
		}
	} else {
		fat_int := fs.GetFATInt()
		if err := fat_int.WriteFAT(disk_ref); err != nil {
			return err
		}
		if err := fat_int.WriteFAT(disk_ref); err != nil {
			return err
		}
	}

	return nil
}
