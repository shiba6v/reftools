package thirdparty

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/packages"
)

func Wrap(err error) error {
	_, file, line, _ := runtime.Caller(1)
	return fmt.Errorf("%s:%d: %w", file, line, err)
}

func FindPos(lprog []*packages.Package, path string, off int) (*ast.File, *packages.Package, token.Pos, error) {
	for _, pkg := range lprog {
		for _, f := range pkg.Syntax {
			if file := pkg.Fset.File(f.Pos()); file.Name() == path {
				if off > file.Size() {
					return nil, nil, 0,
						fmt.Errorf("file size (%d) is smaller than given offset (%d)",
							file.Size(), off)
				}
				return f, pkg, file.Pos(off), nil
			}
		}
	}

	return nil, nil, 0, fmt.Errorf("could not find file %q", path)
}

func allowErrors(lconf *loader.Config) {
	ctxt := *lconf.Build
	ctxt.CgoEnabled = false
	lconf.Build = &ctxt
	lconf.AllowErrors = true
	lconf.ParserMode = parser.AllErrors
	lconf.TypeChecker.Error = func(error) {}
}

func findPos(lprog []*packages.Package, path string, off int) (*ast.File, *packages.Package, token.Pos, error) {
	for _, pkg := range lprog {
		for _, f := range pkg.Syntax {
			if file := pkg.Fset.File(f.Pos()); file.Name() == path {
				if off > file.Size() {
					return nil, nil, 0,
						fmt.Errorf("file size (%d) is smaller than given offset (%d)",
							file.Size(), off)
				}
				return f, pkg, file.Pos(off), nil
			}
		}
	}

	return nil, nil, 0, fmt.Errorf("could not find file %q", path)
}

// func Load(path string, modified bool) (*loader.Program, error) {
// 	ctx := &build.Default
// 	// if modified {
// 	// 	archive, err := buildutil.ParseOverlayArchive(os.Stdin)
// 	// 	if err != nil {
// 	// 		return nil, Wrap(err)
// 	// 	}
// 	// 	ctx = buildutil.OverlayContext(ctx, archive)
// 	// }
// 	// cwd, err := os.Getwd()
// 	// if err != nil {
// 	// 	return nil, Wrap(err)
// 	// }
// 	// pkg, err := buildutil.ContainingPackage(ctx, cwd, path)
// 	// if err != nil {
// 	// 	return nil, Wrap(err)
// 	// }

// 	var btags buildutil.TagsFlag

// 	var overlay map[string][]byte
// 	if modified {
// 		var err error
// 		overlay, err = buildutil.ParseOverlayArchive(os.Stdin)
// 		if err != nil {
// 			log.Fatalf("invalid archive: %v", err)
// 		}
// 	}
// 	cfg := &packages.Config{
// 		Overlay:    overlay,
// 		Mode:       packages.LoadAllSyntax,
// 		Tests:      true,
// 		Dir:        filepath.Dir(path),
// 		Fset:       token.NewFileSet(),
// 		BuildFlags: []string{"-tags", strings.Join([]string(btags), ",")},
// 		Env:        os.Environ(),
// 	}
// 	pkgs, err := packages.Load(cfg)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	f, pkg, pos, err := findPos(pkgs, path, offset)
// 	if err != nil {
// 		return err
// 	}

// 	conf := &loader.Config{Build: ctx}
// 	allowErrors(conf)
// 	conf.ImportWithTests(pkg.ImportPath)

// 	_, rev, _ := importgraph.Build(ctx)
// 	for p := range rev.Search(pkg.ImportPath) {
// 		conf.ImportWithTests(p)
// 	}
// 	return conf.Load()
// }
