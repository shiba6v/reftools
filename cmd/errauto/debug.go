package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os"
)

func debugPrintf(format string, a ...interface{}) {
	f, err := os.OpenFile("/tmp/log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write([]byte(fmt.Sprintf(format, a...))); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func debugAstPrint(x interface{}) {
	f, err := os.OpenFile("/tmp/log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	ast.Fprint(f, token.NewFileSet(), x, ast.NotNilFilter)
}
