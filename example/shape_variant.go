// Code generated by govariant example.go; DO NOT EDIT.
package main

import ()

type Shape interface {
	String() string
	Visit(ShapeVisitor)
	isShape()
}

type ShapeVisitor struct {
	Square    func(float64)
	Rectangle func(float64, float64)
	Circle    func(float64)
}

type Square struct {
	length float64
}

type Rectangle struct {
	length float64
	width  float64
}

type Circle struct {
	radius float64
}

func (Square) isShape()                 {}
func (rcv Square) Unpack() float64      { return rcv.length }
func (rcv Square) Visit(v ShapeVisitor) { v.Square(rcv.length) }

func (Rectangle) isShape()                       {}
func (rcv Rectangle) Unpack() (float64, float64) { return rcv.length, rcv.width }
func (rcv Rectangle) Visit(v ShapeVisitor)       { v.Rectangle(rcv.length, rcv.width) }

func (Circle) isShape()                 {}
func (rcv Circle) Unpack() float64      { return rcv.radius }
func (rcv Circle) Visit(v ShapeVisitor) { v.Circle(rcv.radius) }

var (
	_ Shape = struct{ Square }{}
	_ Shape = struct{ Rectangle }{}
	_ Shape = struct{ Circle }{}
)
