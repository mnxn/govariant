package main

import (
	"fmt"
	"math"
)

//go:generate govariant $GOFILE

// @variant Shape
type _ interface {
	Square(length float64)
	Rectangle(length float64, width float64)
	Circle(radius float64)

	String() string // @method
}

// using methods to distinguish variants

func (s Square) String() string {
	return fmt.Sprintf("Square with length %f", s.length)
}

func (r Rectangle) String() string {
	return fmt.Sprintf("Rectangle with length %f and width %f", r.length, r.width)
}

func (c Circle) String() string {
	return fmt.Sprintf("Circle with radius %f", c.radius)
}

func main() {
	var shape Shape = Rectangle{length: 3, width: 5}

	fmt.Println(shape) // Rectangle with length 3.000000 and width 5.000000

	// using type switch to distinguish variants
	var area float64
	switch s := shape.(type) {
	case Square:
		length := s.Unpack()
		area = length * length
	case Rectangle:
		length, width := s.Unpack()
		area = length * width
	case Circle:
		radius := s.Unpack()
		area = math.Pi * radius * radius
	}
	fmt.Println("Area", area) // Area 15

	// using visitor to distinguish variants
	var perimeter float64
	shape.Visit(ShapeVisitor{
		Square: func(length float64) {
			perimeter = 4 * length
		},
		Rectangle: func(length float64, width float64) {
			perimeter = 2*length + 2*width
		},
		Circle: func(radius float64) {
			perimeter = 2 * math.Pi * radius
		},
	})
	fmt.Println("Perimeter", perimeter) // Perimeter 16
}
