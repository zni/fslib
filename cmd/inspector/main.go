package main

import (
	"flag"
	"os"

	"github.com/zni/fslib/internal/utilities"
	fat32 "github.com/zni/fslib/pkg/fat32"
)

func main() {
	flagset := flag.NewFlagSet("inspector", flag.ExitOnError)
	disk := flagset.String("disk", "", "the disk to inspect")
	if err := flagset.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if *disk == "" {
		utilities.DisplayUsage(flagset)
	}

	fs, err := fat32.Load(*disk)
	if err != nil {
		utilities.HandleError(err)
	}

	fs.PrintInfo()

	if err := fs.Close(); err != nil {
		utilities.HandleError(err)
	}
}
