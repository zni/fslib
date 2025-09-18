package fat

import (
	"errors"
	"io"
	"os"

	"github.com/zni/fslib/internal/utilities"
)

type BPB12 BPB[ExtBPBMinimal]

type ExtBPBMinimal struct {
	bs_drvnum     byte
	bs_reserved1  byte
	bs_bootsig    byte
	bs_volid      uint32
	bs_vollab     [11]byte
	bs_filsystype [8]byte

	// Padded for 448 bytes with value 0x00

	signature_word [2]byte

	// Pad out sector with 0x00
	// Only for media where bpb_bytspersec > 512
}

type BPB16 BPB[ExtBPBMinimal]

type BPB32 BPB[ExtBPBFull]

type ExtBPBFull struct {
	BPB_fatsz32   uint32
	BPB_extflags  uint16
	BPB_fsver     uint16
	BPB_rootclus  uint32
	BPB_fsinfo    uint16
	BPB_bkbootsec uint16
	BPB_reserved  [12]byte
	BS_drvnum     byte
	BS_reserved1  byte
	BS_bootsig    byte
	BS_volid      uint32
	BS_vollab     [11]byte
	BS_filsystype [8]byte

	// 420 pad 0x00 bytes erryday

	signature_word [2]byte
}

func ReadBPB32(f *os.File) (*BPB32, error) {
	var bpb, err = ReadCommonBPB(f)
	if err != nil {
		return nil, err
	}

	short_ := make([]byte, 2)
	byte_ := make([]byte, 1)
	int_ := make([]byte, 4)

	var extbpb ExtBPBFull = ExtBPBFull{}
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
	extbpb.BS_drvnum = byte_[0]

	_, err = f.Seek(1, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	extbpb.BS_bootsig = byte_[0]

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	extbpb.BS_volid = utilities.BytesToInt(int_)

	_, err = f.Read(extbpb.BS_vollab[:])
	if err != nil {
		return nil, err
	}

	_, err = f.Read(extbpb.BS_filsystype[:])
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

	return &BPB32{bpb, &extbpb}, nil
}
