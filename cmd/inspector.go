package main

import (
	"flag"
	"fmt"
	"os"

	fat32 "github.com/zni/fslib/pkg/fat32"
)

func main() {
	flagset := flag.NewFlagSet("disk", flag.ExitOnError)
	disk := flagset.String("disk", "", "the disk to inspect")
	if err := flagset.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if *disk == "" {
		flagset.Usage()
		os.Exit(1)
	}

	fs, err := fat32.Load(*disk)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("error: %v\n", err))
		os.Exit(1)
	}

	fs.PrintInfo()

	if err := fs.Close(); err != nil {
		os.Stderr.WriteString(fmt.Sprintf("error: %v\n", err))
		os.Exit(1)
	}
}
