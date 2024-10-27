# fslib

**FOR THE LOVE OF `DIETY`, DON'T USE THIS ON THINGS YOU VALUE**

This is a project for myself to learn how filesystems work and also to get some practice writing in Go.

## Overview

With the above out of the way, how does this stuff work?

Well, you could use the `pkg/fat32` library portion directly, or use one of the debugging CLI tools in `cmd/`.

### Debugging Tools

### fs.fat32.cat

**DANGER**

Reads in a file's complete contents into memory.

```
$ go build -o local ./cmd/fs.fat32.cat
$ local/fs.fat32.cat -disk local/test1.dsk -path /home
=> read in 21 bytes
welcome to the root.
```

### fs.fat32.catmin

Reads as much of the file as you specify.

```
$ go build -o local ./cmd/fs.fat32.catmin
$ local/fs.fat32.catmin -disk local/test1.dsk -path /home -bytes 10
=> read in 10 bytes
welcome to
```

### fs.fat32.inspector

Mostly useless, just debug information for myself.

```
$ go build -o local ./cmd/fs.fat32.inspector
$ local/fs.fat32.inspector -disk local/test1.dsk -path /home
+---------------------+
|  VOLUME DEBUG INFO  |
+---------------------+
\ volume_filename: test1.dsk
\ bytes_per_sector: 512
\ sectors_per_cluster: 1
\ volume_label: NO NAME
\ file_sys_type: FAT32
\ free_clusters: 123024
\ next_free_cluster: 6

+-------------------+
|  FILE DEBUG INFO  |
+-------------------+
\ filename  : home
\ LDIR loc  : 000f4800
\ DIR loc   : 000f4820
\ directory?: false
\ cluster   : 11
\ file size : 21
```

### fs.fat32.mkdir

Creates an empty directory in the volume. Preceeding path must exist.

```
$ go build -o local ./cmd/fs.fat32.mkdir
$ local/fs.fat32.mkdir -disk local/test1.dsk -path /misc/b/z
+-------------------+
|  FILE DEBUG INFO  |
+-------------------+
\ filename  : z
\ LDIR loc  : 000f6040
\ DIR loc   : 000f6060
\ directory?: true
\ cluster   : 6
\ file size : 0
```
