package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zni/fslib/internal/utilities"
	"github.com/zni/fslib/pkg/fat/fat32"
)

func main() {
	flagset := flag.NewFlagSet("fs.fat32.mkdir", flag.ExitOnError)
	disk := flagset.String("disk", "", "the disk to operate on")
	path := flagset.String("path", "", "the file to cat")
	bytes := flagset.Int("bytes", 0, "the minimum number of bytes to read")
	if err := flagset.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if *disk == "" {
		utilities.DisplayUsage(flagset)
	}

	if *path == "" {
		utilities.DisplayUsage(flagset)
	}

	if *bytes == 0 {
		utilities.DisplayUsage(flagset)
	}

	fs, err := fat32.Load(*disk)
	if err != nil {
		utilities.HandleError(err)
	}

	file, err := fs.ReadFile(*path)
	if err != nil {
		utilities.HandleError(err)
	}

	read_buffer := make([]byte, *bytes)
	bytes_read, err := file.Read(read_buffer)
	if err != nil {
		utilities.HandleError(err)
	}

	if bytes_read == 0 {
		os.Stderr.WriteString(fmt.Sprintln("=> read in 0 bytes"))
	} else {
		os.Stderr.WriteString(fmt.Sprintf("=> read in %d bytes\n", bytes_read))
		fmt.Printf("%s", read_buffer)
	}

	if err := fs.Close(); err != nil {
		utilities.HandleError(err)
	}
}
