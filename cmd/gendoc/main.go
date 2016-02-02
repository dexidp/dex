package main

import (
	"fmt"
	"io"
	"os"

	"github.com/coreos/dex/pkg/gendoc"
	"github.com/spf13/cobra"
)

var cmd = &cobra.Command{
	Use:   "gendoc",
	Short: "Generate documentation from REST specifications.",
	Long:  `A tool to generate documentation for dex's REST APIs.`,
	RunE:  gen,
}

var (
	infile      string
	outfile     string
	readFlavor  string
	writeFlavor string
)

func init() {
	cmd.PersistentFlags().StringVar(&infile, "f", "", "File to read from. If ommitted read from stdin.")
	cmd.PersistentFlags().StringVar(&outfile, "o", "", "File to write to. If ommitted write to stdout.")
	cmd.PersistentFlags().StringVar(&readFlavor, "r", "googleapi", "Flavor of REST spec to read. Currently only supports 'googleapi'.")
	cmd.PersistentFlags().StringVar(&writeFlavor, "w", "markdown", "Flavor of documentation. Currently only supports 'markdown'.")

}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func gen(cmd *cobra.Command, args []string) error {
	var (
		in     io.Reader
		out    io.Writer
		decode func(io.Reader) (gendoc.Document, error)
		encode func(gendoc.Document) ([]byte, error)
	)

	switch readFlavor {
	case "googleapi":
		decode = gendoc.ParseGoogleAPI
	default:
		return fmt.Errorf("unsupported read flavor %q", readFlavor)
	}

	switch writeFlavor {
	case "markdown":
		encode = gendoc.Document.MarshalMarkdown
	default:
		return fmt.Errorf("unsupported write flavor %q", writeFlavor)
	}

	if infile == "" {
		in = os.Stdin
	} else {
		f, err := os.OpenFile(infile, os.O_RDONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}

	if outfile == "" {
		out = os.Stdout
	} else {
		f, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	doc, err := decode(in)
	if err != nil {
		return fmt.Errorf("failed to decode input: %v", err)
	}
	data, err := encode(doc)
	if err != nil {
		return fmt.Errorf("failed to encode document: %v", err)
	}

	_, err = out.Write(data)
	return err
}
