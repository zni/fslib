package common

type BPB struct {
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
