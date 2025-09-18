package fat

import (
	"os"
)

type FATSystem interface {
	FAT12 | FAT16 | FAT32 | *FAT12 | *FAT16 | *FAT32
	GetCommonBPB() *CommonBPB
	GetExtendedBPBMin() *ExtBPBMinimal
	GetExtendedBPBFull() *ExtBPBFull
	GetDiskRef() *os.File
	GetFSInfo() *FSInfo
	GetFATShort() *FAT[uint16]
	GetFATInt() *FAT[uint32]
}

type FAT12 struct {
	BPB       *BPB12
	BackupBPB *BPB12
	FAT       *FAT[uint16]
	BackupFAT *FAT[uint16]
	DiskRef   *os.File
}

func (fs FAT12) GetCommonBPB() *CommonBPB {
	return fs.BPB.Common
}

func (fs FAT12) GetExtendedBPBMin() *ExtBPBMinimal {
	return fs.BPB.Extended
}

func (fs FAT12) GetExtendedBPBFull() *ExtBPBFull {
	return nil
}

func (fs FAT12) GetDiskRef() *os.File {
	return fs.DiskRef
}

func (fs FAT12) GetFSInfo() *FSInfo {
	return nil
}

func (fs FAT12) GetFATShort() *FAT[uint16] {
	return fs.FAT
}

func (fs FAT12) GetFATInt() *FAT[uint32] {
	return nil
}

type FAT16 struct {
	BPB       *BPB16
	BackupBPB *BPB16
	FAT       *FAT[uint16]
	BackupFAT *FAT[uint16]
	DiskRef   *os.File
}

func (fs FAT16) GetCommonBPB() *CommonBPB {
	return fs.BPB.Common
}

func (fs FAT16) GetExtendedBPBMin() *ExtBPBMinimal {
	return fs.BPB.Extended
}

func (fs FAT16) GetExtendedBPBFull() *ExtBPBFull {
	return nil
}

func (fs FAT16) GetDiskRef() *os.File {
	return fs.DiskRef
}

func (fs FAT16) GetFSInfo() *FSInfo {
	return nil
}

func (fs FAT16) GetFATShort() *FAT[uint16] {
	return fs.FAT
}

func (fs FAT16) GetFATInt() *FAT[uint32] {
	return nil
}

type FAT32 struct {
	BPB          *BPB32
	FSInfo       *FSInfo
	BackupBPB    *BPB32
	BackupFSInfo *FSInfo
	FAT          *FAT[uint32]
	BackupFAT    *FAT[uint32]
	DiskRef      *os.File
}

func (vol FAT32) GetCommonBPB() *CommonBPB {
	return vol.BPB.Common
}

func (vol FAT32) GetExtendedBPBMin() *ExtBPBMinimal {
	return nil
}

func (vol FAT32) GetExtendedBPBFull() *ExtBPBFull {
	return vol.BPB.Extended
}

func (vol FAT32) GetDiskRef() *os.File {
	return vol.DiskRef
}

func (vol FAT32) GetFSInfo() *FSInfo {
	return vol.FSInfo
}

func (vol FAT32) GetFATShort() *FAT[uint16] {
	return nil
}

func (vol FAT32) GetFATInt() *FAT[uint32] {
	return vol.FAT
}
