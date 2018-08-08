package main

import "fmt"
import "math"

func main() {
	fmt.Println("Hello, world")

	var age int
	fmt.Println("age is", age)

    foo := 5
	fmt.Println("foo is", foo)

	var bar = 3
	fmt.Println("bar is", bar)

	yPtr := new(int)
	println("yPtr is", yPtr)

	// Don't need to use it.
	const pi = 3.14159

	i := 1
	for i <= 5 {
		i += 1
	}

	for j := 0; j <= 5; j++ {

	}

	var favNums[5] float64
	fmt.Println(favNums)

	// Or [5]float64 ...
	favNums3 := []float64 {10, 20, 30, 40, 50}

	for i, value := range favNums3 {
		fmt.Println(i, value)
	}

	// WTF. Why does it quietly ignore that 'range' returns
	// a tuple?
	for i := range favNums3 {
		fmt.Println(i)
	}

	// Five zeros, capacity 10
	numSlice := make([]float64, 5, 10)
	fmt.Println(numSlice)
	copy(numSlice, favNums3)
	fmt.Println(numSlice)
	numSlice = append(numSlice, 60, 70)
	fmt.Println(numSlice)


	presAge := make(map[string] int)
	presAge["TRoosevelt"] = 42
	fmt.Println(presAge)
	delete(presAge, "TRoosevelt")
	delete(presAge, "blarg")

	x := 1
	doubleNum := func(y int) int {
		return y * 2
	}
	fmt.Println(doubleNum(x))

	saveDiv(4, 0)
	saveDiv(4, 2)

	// WTF: println includes a space after "is", but print doesn't.
	print("c() is", c())
	println()
	println("c() is", c())

	changeVal(&x, 5)
	println("x is now", x)

	rect1 := Rectangle{x: 1, y: 2, height: 3, width: 4}
	rect2 := Rectangle{1, 2, 3, 4}

	println(rect1.area())
	println(rect2.area())

	circ := Circle{radius: 3}
	printShapeArea(&circ)
}

func changeVal(x *int, val int) {
	*x = val
}

func saveDiv(num1, num2 int) int {
	defer func() {
		fmt.Println("Recover: ", recover())
	}()

	return num1 / num2
}

func c() (i int) {
    defer func() { i++ }()
    return 1
}

type Rectangle struct {
	x float64
	y float64
	height float64
	width float64
}

func (rect *Rectangle) area() float64 {
	return rect.width * rect.height
}

func (rect Rectangle) area2() float64 {
	return rect.width * rect.height
}


type Shape interface {
	area() float64
}

type Circle struct {
	radius float64
}

// Cannot take *Circle, or else
// "Circle does not implement Shape (area method has pointer receiver)"
func (circle Circle) area() float64 {
	return math.Pi * circle.radius * circle.radius
}

func printShapeArea(shape *Shape) {
	println("Area is ", (*shape).area())
}