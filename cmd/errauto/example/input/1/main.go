package main

import "github.com/shiba6v/reftools/cmd/errauto/example/input/1/somepkg"

type param2 struct {
	P int
}

type input1 struct {
	Param1 string
	Param2 param2
}

type result1 struct {
	Param1 int
	Param2 param2
}

func func1() (input1, somepkg.PkgInput2, int, error) {
	return input1{}, somepkg.PkgInput2{}, 0, nil
}

func run() (result1, somepkg.PkgOutput2, somepkg.PkgOutput3, string, error) {
	i1, i2, t, err := func1()
	
	_ = i1
	_ = i2
	_ = t
	_ = err
	return result1{}, somepkg.PkgOutput2{}, 0, "", nil
}

func main() {
	_, _, _, _, _ = run()
}
