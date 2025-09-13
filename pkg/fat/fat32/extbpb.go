package fat32

import (
	"errors"
	"io"
	"os"

	"github.com/zni/fslib/internal/utilities"
	"github.com/zni/fslib/pkg/fat/common"
)

type BPB32 struct {
	Common   common.BPB
	Extended ExtBPB32
}

type ExtBPB32 struct {
	BPB_fatsz32   uint32
	BPB_extflags  uint16
	BPB_fsver     uint16
	BPB_rootclus  uint32
	BPB_fsinfo    uint16
	BPB_bkbootsec uint16
	BPB_reserved  [12]byte
	bs_drvum      byte
	bs_reserved1  byte
	bs_bootsig    byte
	bs_volid      uint32
	bs_vollab     [11]byte
	bs_filsystype [8]byte

	// 420 pad 0x00 bytes erryday

	signature_word [2]byte
}

func readBPB(f *os.File) (*BPB32, error) {
	var bpb common.BPB = common.BPB{}
	short_ := make([]byte, 2)
	byte_ := make([]byte, 1)
	int_ := make([]byte, 4)

	_, err := io.ReadFull(f, bpb.BS_jmpboot[:])
	if err != nil {
		return nil, err
	}

	_, err = io.ReadFull(f, bpb.BS_oemname[:])
	if err != nil {
		return nil, err
	}

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_bytspersec = utilities.BytesToShort(short_)

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_secperclus = byte_[0]

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_rsvdseccnt = utilities.BytesToShort(short_)

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_numfats = byte_[0]

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_rootentcnt = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_totsec16 = utilities.BytesToShort(short_)

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_media = byte_[0]

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_fatsz16 = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_secpertrk = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_numheads = utilities.BytesToShort(short_)

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_hiddsec = utilities.BytesToInt(int_)

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	bpb.BPB_totsec32 = utilities.BytesToInt(int_)

	var extbpb ExtBPB32 = ExtBPB32{}
	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	extbpb.BPB_fatsz32 = utilities.BytesToInt(int_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	extbpb.BPB_extflags = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	extbpb.BPB_fsver = utilities.BytesToShort(short_)

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	extbpb.BPB_rootclus = utilities.BytesToInt(int_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	extbpb.BPB_fsinfo = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	extbpb.BPB_bkbootsec = utilities.BytesToShort(short_)

	_, err = f.Seek(12, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	extbpb.bs_drvum = byte_[0]

	_, err = f.Seek(1, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	extbpb.bs_bootsig = byte_[0]

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	extbpb.bs_volid = utilities.BytesToInt(int_)

	_, err = f.Read(extbpb.bs_vollab[:])
	if err != nil {
		return nil, err
	}

	_, err = f.Read(extbpb.bs_filsystype[:])
	if err != nil {
		return nil, err
	}

	_, err = f.Seek(420, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(extbpb.signature_word[:])
	if err != nil {
		return nil, err
	}

	if extbpb.signature_word[0] != 0x55 && extbpb.signature_word[1] != 0xAA {
		return nil, errors.New("invalid BPB signature")
	}

	return &BPB32{bpb, extbpb}, nil
}
