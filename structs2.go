package main

import (
	"fmt"
)

type Fooer interface {
	Foo()
}

type FooImpl struct{}

/*
./structs2.go:19: cannot use f1 (type FooImpl) as type Fooer in argument to DoFoo:
	FooImpl does not implement Fooer (Foo method has pointer receiver)
*/
//func (f *FooImpl) Foo() {}

func (f FooImpl) Foo() {}

func main() {
	var f1 FooImpl
	f1.Foo()
	DoFoo(f1)
}

/*
./structs2.go:24: cannot use f1 (type FooImpl) as type *Fooer in argument to DoFoo:
	*Fooer is pointer to interface, not interface

or:

./structs2.go:24: cannot use &f1 (type *FooImpl) as type *Fooer in argument to DoFoo:
	*Fooer is pointer to interface, not interface

func DoFoo(f *Fooer) {
	fmt.Printf("Type [%T] %+v\n", f, f)
}
*/

func DoFoo(f Fooer) {
	fmt.Printf("Type [%T] %+v\n", f, f)
}
