package fat32

import (
	"errors"
	"io"
	"os"

	"github.com/zni/fslib/internal/utilities"
)

const lead_signature uint32 = 0x41615252
const structure_signature uint32 = 0x61417272
const trailing_signature uint32 = 0xAA550000

type FSInfo struct {
	lead_sig   uint32
	struc_sig  uint32
	free_count uint32
	next_free  uint32
	trail_sig  uint32
}

func readFSInfo(f *os.File) (*FSInfo, error) {
	var fsinfo FSInfo
	int_ := make([]uint8, 4)

	_, err := f.Read(int_)
	if err != nil {
		return nil, err
	}
	fsinfo.lead_sig = utilities.BytesToInt(int_)
	if fsinfo.lead_sig != lead_signature {
		return nil, errors.New(`invalid lead signature`)
	}

	_, err = f.Seek(480, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	fsinfo.struc_sig = utilities.BytesToInt(int_)
	if fsinfo.struc_sig != structure_signature {
		return nil, errors.New(`invalid structure signature`)
	}

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	fsinfo.free_count = utilities.BytesToInt(int_)

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	fsinfo.next_free = utilities.BytesToInt(int_)

	_, err = f.Seek(12, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	fsinfo.trail_sig = utilities.BytesToInt(int_)
	if fsinfo.trail_sig != trailing_signature {
		return nil, errors.New(`invalid trailing signature`)
	}

	return &fsinfo, nil
}

func seekToFSInfo(fs *FAT32) error {
	if _, err := fs.file.Seek(int64(fs.BPB.Common.bpb_bytspersec), io.SeekStart); err != nil {
		return err
	}

	return nil
}

func writeFSInfo(fs *FAT32) error {
	if err := seekToFSInfo(fs); err != nil {
		return err
	}

	_, err := fs.file.Write(utilities.IntToBytes(fs.FSInfo.lead_sig))
	if err != nil {
		return err
	}

	_, err = fs.file.Seek(480, io.SeekCurrent)
	if err != nil {
		return err
	}

	_, err = fs.file.Write(utilities.IntToBytes(fs.FSInfo.struc_sig))
	if err != nil {
		return err
	}

	_, err = fs.file.Write(utilities.IntToBytes(fs.FSInfo.free_count))
	if err != nil {
		return err
	}

	_, err = fs.file.Write(utilities.IntToBytes(fs.FSInfo.next_free))
	if err != nil {
		return err
	}

	_, err = fs.file.Seek(12, io.SeekCurrent)
	if err != nil {
		return err
	}

	_, err = fs.file.Write(utilities.IntToBytes(fs.FSInfo.trail_sig))
	if err != nil {
		return err
	}

	return nil
}
