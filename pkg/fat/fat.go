package fat

import (
	"errors"
	"io"
	"os"

	"github.com/zni/fslib/internal/utilities"
)

type FATSize interface {
	~uint16 | ~uint32
}

type FAT[T FATSize] struct {
	table []T
}

func MakeFAT16(max_clusters uint32) *FAT[uint16] {
	fat := make([]uint16, max_clusters)

	return &FAT[uint16]{fat}
}

func MakeFAT32(max_clusters uint32) *FAT[uint32] {
	fat := make([]uint32, max_clusters)

	return &FAT[uint32]{fat}
}

func (fat *FAT[T]) ReadFAT(fs *os.File, max_clusters uint32) error {
	if table, ok := any(fat.table).([]uint16); ok {
		return readFAT16(fs, max_clusters, table)
	} else if table, ok := any(fat.table).([]uint32); ok {
		return readFAT32(fs, max_clusters, table)
	}
	return nil
}

func readFAT16(f *os.File, max_clusters uint32, table []uint16) error {
	short_ := make([]uint8, 2)

	var n uint32
	for n = 0; n < max_clusters; n++ {
		_, err := f.Read(short_)
		if err != nil {
			return errors.New("failed to read cluster entry")
		}
		table[n] = utilities.BytesToShort(short_)
	}

	return nil
}

func readFAT32(f *os.File, max_clusters uint32, table []uint32) error {
	int_ := make([]uint8, 4)

	var n uint32
	for n = 0; n < max_clusters; n++ {
		_, err := f.Read(int_)
		if err != nil {
			return errors.New("failed to read cluster entry")
		}
		table[n] = utilities.BytesToInt(int_)
	}

	return nil
}

func (fat *FAT[T]) WriteFAT(fs *os.File) error {
	if table, ok := any(fat.table).([]uint16); ok {
		return writeFAT16(fs, table)
	} else if table, ok := any(fat.table).([]uint32); ok {
		return writeFAT32(fs, table)
	}

	return nil
}

func writeFAT16(fs *os.File, table []uint16) error {
	for _, v := range table {
		if _, err := fs.Write(utilities.ShortToBytes(v)); err != nil {
			return err
		}
	}

	for _, v := range table {
		if _, err := fs.Write(utilities.ShortToBytes(v)); err != nil {
			return err
		}
	}

	return nil
}

func writeFAT32(fs *os.File, table []uint32) error {
	for _, v := range table {
		if _, err := fs.Write(utilities.IntToBytes(v)); err != nil {
			return err
		}
	}

	for _, v := range table {
		if _, err := fs.Write(utilities.IntToBytes(v)); err != nil {
			return err
		}
	}

	return nil
}

func SeekToFAT(fs *os.File, bpb *CommonBPB) error {
	var fat_loc int64 = int64(bpb.BPB_rsvdseccnt) * int64(bpb.BPB_bytspersec)
	if _, err := fs.Seek(fat_loc, io.SeekStart); err != nil {
		return err
	}

	return nil
}

/*
Get the next free cluster from the FAT not marked EOC.
*/
func (fat *FAT[T]) GetNextFreeCluster() (T, error) {
	for i := 2; i < len(fat.table); i++ {
		if fat.table[i] == 0 {
			return T(i), nil
		}
	}

	return 0, errors.New("no free clusters")
}

/*
Mark a cluster in the FAT with the EOC value.
*/
func (fat *FAT[T]) MarkEOC(cluster uint) {
	fat.table[cluster] = fat.table[1]
}

/*
Get the cluster EOC value.
*/
func (fat *FAT[T]) GetEOC() T {
	return fat.table[1]
}

/*
Get the cluster at the specified location.
*/
func (fat *FAT[T]) GetCluster(loc uint) T {
	return fat.table[loc]
}
