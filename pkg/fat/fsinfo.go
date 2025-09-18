package fat

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

func (fsinfo *FSInfo) Read(f *os.File) error {
	int_ := make([]uint8, 4)

	_, err := f.Read(int_)
	if err != nil {
		return err
	}
	fsinfo.lead_sig = utilities.BytesToInt(int_)
	if fsinfo.lead_sig != lead_signature {
		return errors.New(`invalid lead signature`)
	}

	_, err = f.Seek(480, io.SeekCurrent)
	if err != nil {
		return err
	}

	_, err = f.Read(int_)
	if err != nil {
		return err
	}
	fsinfo.struc_sig = utilities.BytesToInt(int_)
	if fsinfo.struc_sig != structure_signature {
		return errors.New(`invalid structure signature`)
	}

	_, err = f.Read(int_)
	if err != nil {
		return err
	}
	fsinfo.free_count = utilities.BytesToInt(int_)

	_, err = f.Read(int_)
	if err != nil {
		return err
	}
	fsinfo.next_free = utilities.BytesToInt(int_)

	_, err = f.Seek(12, io.SeekCurrent)
	if err != nil {
		return err
	}

	_, err = f.Read(int_)
	if err != nil {
		return err
	}
	fsinfo.trail_sig = utilities.BytesToInt(int_)
	if fsinfo.trail_sig != trailing_signature {
		return errors.New(`invalid trailing signature`)
	}

	return nil
}

func seekToFSInfo(fs *os.File, bpb *CommonBPB) error {
	if _, err := fs.Seek(int64(bpb.BPB_bytspersec), io.SeekStart); err != nil {
		return err
	}

	return nil
}

func (fsinfo *FSInfo) Write(fs *os.File, bpb *CommonBPB) error {
	if err := seekToFSInfo(fs, bpb); err != nil {
		return err
	}

	_, err := fs.Write(utilities.IntToBytes(fsinfo.lead_sig))
	if err != nil {
		return err
	}

	_, err = fs.Seek(480, io.SeekCurrent)
	if err != nil {
		return err
	}

	_, err = fs.Write(utilities.IntToBytes(fsinfo.struc_sig))
	if err != nil {
		return err
	}

	_, err = fs.Write(utilities.IntToBytes(fsinfo.free_count))
	if err != nil {
		return err
	}

	_, err = fs.Write(utilities.IntToBytes(fsinfo.next_free))
	if err != nil {
		return err
	}

	_, err = fs.Seek(12, io.SeekCurrent)
	if err != nil {
		return err
	}

	_, err = fs.Write(utilities.IntToBytes(fsinfo.trail_sig))
	if err != nil {
		return err
	}

	return nil
}
