package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/betalo-sweden/moq/pkg/moq"
)

func main() {
	log.SetPrefix("")
	log.SetFlags(0)

	var (
		outFile = flag.String("out", "", "output file (default stdout)")
		pkgName = flag.String("pkg", "", "package name (default will infer)")
	)
	flag.Usage = func() {
		fmt.Println(`moq [flags] destination interface [interface2 [interface3 [...]]]`)
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "not enough arguments")
		flag.Usage()
		os.Exit(1)
	}

	destination := args[0]
	args = args[1:]

	// setup mock context
	m, err := moq.New(destination, *pkgName)
	if err != nil {
		log.Fatalln(err)
	}

	var buf bytes.Buffer
	var out io.Writer
	out = os.Stdout
	if len(*outFile) > 0 {
		out = &buf
	}

	// generate mock source
	err = m.Mock(out, args...)
	if err != nil {
		log.Fatalln(err)
	}

	// create the file
	if len(*outFile) > 0 {
		err = ioutil.WriteFile(*outFile, buf.Bytes(), 0644)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
