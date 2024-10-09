package fat32

import (
	"errors"
	"io"
	"os"

	"github.com/zni/fslib/internal/utilities"
)

type BPB struct {
	jmp_boot []uint8
	oem_name []uint8

	bytes_per_sector      uint16
	sectors_per_cluster   uint8
	reserved_sector_count uint16
	number_of_fats        uint8
	root_ent_cnt          uint16
	total_sectors_16      uint16
	media                 uint8
	fat_size_16           uint16
	sectors_per_track     uint16
	num_heads             uint16
	hidden_sectors        uint32
	total_sectors_32      uint32

	fat_size_32  uint32
	extflags     uint16
	fs_version   uint16
	root_cluster uint32
	fsinfo       uint16
	bkbootsec    uint16

	drive_num     uint8
	boot_sig      uint8
	volume_id     uint32
	volume_label  []uint8
	file_sys_type []uint8

	signature_word []uint8
}

func readBPB(f *os.File) (*BPB, error) {
	var bpb BPB = BPB{}
	bpb.jmp_boot = make([]uint8, 3)
	bpb.oem_name = make([]uint8, 8)
	bpb.volume_label = make([]uint8, 11)
	bpb.file_sys_type = make([]uint8, 8)
	bpb.signature_word = make([]uint8, 2)

	short_ := make([]uint8, 2)
	byte_ := make([]uint8, 1)
	int_ := make([]uint8, 4)

	_, err := f.Read(bpb.jmp_boot)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(bpb.oem_name)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.bytes_per_sector = utilities.BytesToShort(short_)

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	bpb.sectors_per_cluster = byte_[0]

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.reserved_sector_count = utilities.BytesToShort(short_)

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	bpb.number_of_fats = byte_[0]

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.root_ent_cnt = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.total_sectors_16 = utilities.BytesToShort(short_)

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	bpb.media = byte_[0]

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.fat_size_16 = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.sectors_per_track = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.num_heads = utilities.BytesToShort(short_)

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	bpb.hidden_sectors = utilities.BytesToInt(int_)

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	bpb.total_sectors_32 = utilities.BytesToInt(int_)

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	bpb.fat_size_32 = utilities.BytesToInt(int_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.extflags = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.fs_version = utilities.BytesToShort(short_)

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	bpb.root_cluster = utilities.BytesToInt(int_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.fsinfo = utilities.BytesToShort(short_)

	_, err = f.Read(short_)
	if err != nil {
		return nil, err
	}
	bpb.bkbootsec = utilities.BytesToShort(short_)

	_, err = f.Seek(12, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	bpb.drive_num = byte_[0]

	_, err = f.Seek(1, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(byte_)
	if err != nil {
		return nil, err
	}
	bpb.boot_sig = byte_[0]

	_, err = f.Read(int_)
	if err != nil {
		return nil, err
	}
	bpb.volume_id = utilities.BytesToInt(int_)

	_, err = f.Read(bpb.volume_label)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(bpb.file_sys_type)
	if err != nil {
		return nil, err
	}

	_, err = f.Seek(420, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(bpb.signature_word)
	if err != nil {
		return nil, err
	}

	if bpb.signature_word[0] != 0x55 && bpb.signature_word[1] != 0xAA {
		return nil, errors.New("invalid BPB signature")
	}

	return &bpb, nil
}
