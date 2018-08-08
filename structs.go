package main

import (
	"fmt"
)

type Fooer interface {
	Foo()
}

type FooImpl struct{}

func (f FooImpl) Foo() {}

func main() {
	var f1 FooImpl
	var f2 *FooImpl = &FooImpl{}

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
