package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zni/fslib/internal/utilities"
	"github.com/zni/fslib/pkg/fat32"
)

func main() {
	flagset := flag.NewFlagSet("fs.fat32.mkdir", flag.ExitOnError)
	disk := flagset.String("disk", "", "the disk to operate on")
	path := flagset.String("path", "", "the file to cat")
	if err := flagset.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if *disk == "" {
		utilities.DisplayUsage(flagset)
	}

	if *path == "" {
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

	bytes_read, err := file.Read()
	if err != nil {
		utilities.HandleError(err)
	}

	if bytes_read == 0 {
		os.Stderr.WriteString(fmt.Sprintln("read 0 bytes"))
	} else {
		fmt.Printf("%s\n", file.Content)
	}

	if err := fs.Close(); err != nil {
		utilities.HandleError(err)
	}
}
