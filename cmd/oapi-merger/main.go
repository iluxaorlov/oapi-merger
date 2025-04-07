package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

var (
	flagInputFile    string
	flagOutputFile   string
	flagOutputFormat string
	flagPrintUsage   bool
)

func main() {
	flag.StringVar(&flagInputFile, "i", "", "Input file.")
	flag.StringVar(&flagOutputFile, "o", "", "Where to output merged file, stdout is default.")
	flag.StringVar(&flagOutputFormat, "f", "yaml", "Output format: yaml or json, yaml is default.")
	flag.BoolVar(&flagPrintUsage, "help", false, "Show this help and exit.")
	flag.BoolVar(&flagPrintUsage, "h", false, "Same as -help.")

	flag.Parse()

	if flagPrintUsage {
		flag.Usage()
		os.Exit(0)
	}

	if flagInputFile == "" {
		errExit("input file is required")
	}

	inputFilePath, err := filepath.Abs(flagInputFile)
	if err != nil {
		errExit("error getting input file absolute path: %s", err)
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(inputFilePath)
	if err != nil {
		errExit("error loading file: %s", err)
	}

	doc.InternalizeRefs(loader.Context, func(t *openapi3.T, ref openapi3.ComponentRef) string {
		return strings.TrimSuffix(filepath.Base(ref.RefString()), filepath.Ext(ref.RefString()))
	})

	if err := doc.Validate(loader.Context); err != nil {
		errExit("error validating document: %s", err)
	}

	var b []byte

	switch flagOutputFormat {
	case "yaml":
		b, err = marshalYAML(doc)
	case "json":
		b, err = marshalJSON(doc)
	default:
		errExit("unsupported output format: %s", flagOutputFormat)
	}

	var w io.Writer

	if flagOutputFile == "" {
		w = os.Stdout
	} else {
		outputFilePath, err := filepath.Abs(flagOutputFile)
		if err != nil {
			errExit("error getting output file absolute path: %s", err)
		}

		outputFile, err := os.Create(outputFilePath)
		if err != nil {
			errExit("error creating output file: %s", err)
		}

		defer func() {
			if err := outputFile.Close(); err != nil {
				errExit("error closing output file: %s", err)
			}
		}()

		w = outputFile
	}

	if _, err := w.Write(b); err != nil {
		errExit("error writing output file: %s", err)
	}
}

func marshalYAML(doc *openapi3.T) ([]byte, error) {
	if doc == nil {
		return nil, nil
	}

	x, err := doc.MarshalYAML()
	if err != nil {
		return nil, err
	}

	buffer := &bytes.Buffer{}

	e := yaml.NewEncoder(buffer)
	e.SetIndent(2)

	if err := e.Encode(x); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func marshalJSON(doc *openapi3.T) ([]byte, error) {
	if doc == nil {
		return nil, nil
	}

	x, err := doc.MarshalYAML()
	if err != nil {
		return nil, err
	}

	buffer := &bytes.Buffer{}

	e := json.NewEncoder(buffer)
	e.SetIndent("", "  ")

	if err := e.Encode(x); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func errExit(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}

	_, _ = fmt.Fprintf(os.Stderr, format, args...)

	os.Exit(1)
}
