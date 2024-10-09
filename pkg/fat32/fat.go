package fat32

import (
	"errors"
	"io"
	"os"

	"github.com/zni/fslib/internal/utilities"
)

func readFAT(f *os.File, max_clusters uint32) ([]uint32, error) {
	int_ := make([]uint8, 4)
	fat := make([]uint32, max_clusters)

	var n uint32
	for n = 0; n < max_clusters; n++ {
		_, err := f.Read(int_)
		if err != nil {
			return nil, errors.New("failed to read cluster entry")
		}
		fat[n] = utilities.BytesToInt(int_)
	}

	return fat, nil
}

func writeFAT(fs *FAT32) error {
	for _, v := range fs.FAT {
		if _, err := fs.file.Write(utilities.IntToBytes(v)); err != nil {
			return err
		}
	}

	for _, v := range fs.BackupFAT {
		if _, err := fs.file.Write(utilities.IntToBytes(v)); err != nil {
			return err
		}
	}

	return nil
}

func seekToFAT(fs *FAT32) error {
	var fat_loc int64 = int64(fs.BPB.reserved_sector_count) * int64(fs.BPB.bytes_per_sector)
	if _, err := fs.file.Seek(fat_loc, io.SeekStart); err != nil {
		return err
	}

	return nil
}
