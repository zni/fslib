package fat32

import (
	"errors"
	"io"
	"os"

	"github.com/zni/fslib/internal/utilities"
)

type BPB32 struct {
	Common   BPB
	Extended ExtBPB32
}

// TODO
// Common to all FAT types.
// Move somewhere that is not here.
type BPB struct {
	bs_jmpboot [3]byte
	bs_oemname [8]byte

	bpb_bytspersec uint16
	bpb_secperclus byte
	bpb_rsvdseccnt uint16
	bpb_numfats    byte
	bpb_rootentcnt uint16
	bpb_totsec16   uint16
	bpb_media      byte
	bpb_fatsz16    uint16
	bpb_secpertrk  uint16
	bpb_numheads   uint16
	bpb_hiddsec    uint32
	bpb_totsec32   uint32
}

type ExtBPB32 struct {
	bpb_fatsz32   uint32
	bpb_extflags  uint16
	bpb_fsver     uint16
	bpb_rootclus  uint32
	bpb_fsinfo    uint16
	bpb_bkbootsec uint16
	bpb_reserved  [12]byte
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
	var bpb BPB = BPB{}
	short_ := make([]byte, 2)
	byte_ := make([]byte, 1)
	int_ := make([]byte, 4)

	_, err := io.ReadFull(f, bpb.bs_jmpboot[:])
	if err != nil {
		return nil, err
	}

	_, err = io.ReadFull(f, bpb.bs_oemname[:])
	if err != nil {
		return nil, err
	}

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_bytspersec = utilities.BytesToShort(short_)

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_secperclus = byte_[0]

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_rsvdseccnt = utilities.BytesToShort(short_)

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_numfats = byte_[0]

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_rootentcnt = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_totsec16 = utilities.BytesToShort(short_)

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_media = byte_[0]

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_fatsz16 = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_secpertrk = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_numheads = utilities.BytesToShort(short_)

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_hiddsec = utilities.BytesToInt(int_)

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	bpb.bpb_totsec32 = utilities.BytesToInt(int_)

	var extbpb ExtBPB32 = ExtBPB32{}
	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	extbpb.bpb_fatsz32 = utilities.BytesToInt(int_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	extbpb.bpb_extflags = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	extbpb.bpb_fsver = utilities.BytesToShort(short_)

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	extbpb.bpb_rootclus = utilities.BytesToInt(int_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	extbpb.bpb_fsinfo = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	extbpb.bpb_bkbootsec = utilities.BytesToShort(short_)

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
