package fat12

import "github.com/zni/fslib/pkg/fat/common"

type BPB12 struct {
	Common   common.BPB
	Extended ExtBPB12
}

type ExtBPB12 struct {
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
