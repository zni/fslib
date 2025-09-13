package fat12

type ExtBPB struct {
	bs_drvum      uint8
	bs_reserved1  uint8
	bs_bootsig    uint8
	bs_volid      uint32
	bs_vollab     [11]uint8
	bs_filsystype [8]uint8

	// Padded for 448 bytes with value 0x00

	signature_word uint16

	// Pad out sector with 0x00
	// Only for media where bpb_bytspersec > 512
}
