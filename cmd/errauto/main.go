package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"go/format"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/shiba6v/reftools/cmd/errauto/thirdparty"
	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/packages"
)

type Args struct {
	FileName string
	Offset   int
}

func parseArgs() (Args, error) {
	var (
		filename = flag.String("file", "", "filename")
		offset   = flag.Int("offset", 0, "byte offset")
	)
	flag.Parse()
	// TODO: テンプレート
	return Args{
		FileName: *filename,
		Offset:   *offset,
	}, nil
}

func Output(res Result, dst io.Writer) error {
	fset := token.NewFileSet()
	file := fset.AddFile("", -1, res.Lines)
	for i := 1; i <= res.Lines; i++ {
		file.AddLine(i)
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, res.N); err != nil {
		return Wrap(err)
	}
	type out struct {
		Start int    `json:"start"`
		End   int    `json:"end"`
		Code  string `json:"code"`
	}
	if err := json.NewEncoder(dst).Encode([]out{
		{
			Start: res.Start,
			End:   res.End,
			Code:  buf.String(),
		},
	}); err != nil {
		return Wrap(err)
	}
	return nil
}

func run() error {
	var btags buildutil.TagsFlag

	args, err := parseArgs()
	if err != nil {
		return Wrap(err)
	}
	path, err := filepath.EvalSymlinks(args.FileName)
	if err != nil {
		return Wrap(err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return Wrap(err)
	}
	overlay, err := buildutil.ParseOverlayArchive(os.Stdin)
	if err != nil {
		log.Fatalf("invalid archive: %v", err)
	}
	cfg := &packages.Config{
		Overlay:    overlay,
		Mode:       packages.LoadAllSyntax,
		Tests:      true,
		Dir:        filepath.Dir(path),
		Fset:       token.NewFileSet(),
		BuildFlags: []string{"-tags", strings.Join([]string(btags), ",")},
		Env:        os.Environ(),
	}
	pkgs, err := packages.Load(cfg)
	if err != nil {
		log.Fatal(err)
	}
	f, pkg, pos, err := thirdparty.FindPos(pkgs, path, args.Offset)
	if err != nil {
		return err
	}
	res, err := ErrAuto(Input{
		F:   f,
		Pkg: pkg,
		Pos: pos,
	})
	if err != nil {
		return Wrap(err)
	}
	if err := Output(res, os.Stdout); err != nil {
		return Wrap(err)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
