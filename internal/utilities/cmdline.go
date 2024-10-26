package utilities

import (
	"flag"
	"fmt"
	"os"
)

/*
Display error text and exit with a non-zero code.
*/
func HandleError(err error) {
	os.Stderr.WriteString(fmt.Sprintf("error: %v\n", err))
	os.Exit(1)
}

/*
Display usage information and exit with a non-zero code.
*/
func DisplayUsage(flagset *flag.FlagSet) {
	flagset.Usage()
	os.Exit(1)
}
