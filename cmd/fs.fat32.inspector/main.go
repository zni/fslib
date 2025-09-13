package main

import (
	"flag"
	"os"

	"github.com/zni/fslib/internal/utilities"
	fat32 "github.com/zni/fslib/pkg/fat/fat32"
)

func main() {
	flagset := flag.NewFlagSet("inspector", flag.ExitOnError)
	disk := flagset.String("disk", "", "the disk to inspect")
	filepath := flagset.String("path", "", "the path to inspect")
	if err := flagset.Parse(os.Args[1:]); err != nil {
		utilities.DisplayUsage(flagset)
	}

	if *disk == "" {
		utilities.DisplayUsage(flagset)
	}

	if *filepath == "" {
		utilities.DisplayUsage(flagset)
	}

	fs, err := fat32.Load(*disk)
	if err != nil {
		utilities.HandleError(err)
	}

	fs.PrintInfo()

	file, err := fs.ReadFile(*filepath)
	if err != nil {
		utilities.HandleError(err)
	}

	file.PrintInfo()

	if err := fs.Close(); err != nil {
		utilities.HandleError(err)
	}
}
