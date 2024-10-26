package main

import (
	"flag"
	"os"

	"github.com/zni/fslib/internal/utilities"
	fat32 "github.com/zni/fslib/pkg/fat32"
)

func main() {
	flagset := flag.NewFlagSet("fs.fat32.mkdir", flag.ExitOnError)
	disk := flagset.String("disk", "", "the disk to inspect")
	path := flagset.String("path", "", "the path to create")
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

	_, err = fs.CreateDir(*path)
	if err != nil {
		utilities.HandleError(err)
	}

	if err := fs.Close(); err != nil {
		utilities.HandleError(err)
	}
}
