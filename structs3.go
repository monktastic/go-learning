package main

import (
	"fmt"
)

type Fooer interface {
	Foo()
}

type Foo struct{}

func (f Foo) Foo() {}

func main() {
	var f1 Foo
	var f2 *Foo = &Foo{}

	f1.Foo()
	f2.Foo()
	
	// Because you can invoke Foo() on Foo or *Foo,
	// both are valid `Fooer`s.
	DoFoo(f1)
	DoFoo(&f1)
	DoFoo(f2)
}

func DoFoo(f Fooer) {
	fmt.Printf("Type [%T] %+v\n", f, f)
}
