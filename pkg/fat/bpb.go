package fat

import (
	"io"
	"os"

	"github.com/zni/fslib/internal/utilities"
)

type BPB[T any] struct {
	Common   *CommonBPB
	Extended *T
}

type CommonBPB struct {
	BS_jmpboot [3]byte
	BS_oemname [8]byte

	BPB_bytspersec uint16
	BPB_secperclus byte
	BPB_rsvdseccnt uint16
	BPB_numfats    byte
	BPB_rootentcnt uint16
	BPB_totsec16   uint16
	BPB_media      byte
	BPB_fatsz16    uint16
	BPB_secpertrk  uint16
	BPB_numheads   uint16
	BPB_hiddsec    uint32
	BPB_totsec32   uint32
}

func ReadCommonBPB(f *os.File) (*CommonBPB, error) {
	var bpb CommonBPB = CommonBPB{}
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

	return &bpb, nil
}
