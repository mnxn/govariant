package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	filePath  string
	noCheck   = flag.Bool("nocheck", false, "disable generation of explicit interface implementation checks")
	noUnpack  = flag.Bool("nounpack", false, "disable generation of unpack methods")
	noVisitor = flag.Bool("novisitor", false, "disable generation of visitor struct and methods")
)

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [flags] filepath\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Println("   filepath")
		fmt.Println("    \tpath to file with variant declarations")
	}

	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	filePath = flag.Arg(0)

	variants, err := variantsFromFile(filePath)
	check(err)

	for _, variant := range variants {
		file, err := os.Create(filepath.Join(
			filepath.Dir(filePath),
			strings.ToLower(variant.Name)+"_variant.go"))
		check(err)
		sourceCode, err := generateSourceCode(variant)
		check(err)
		_, err = file.Write(sourceCode)
		check(err)
		err = file.Close()
		check(err)
	}
}
