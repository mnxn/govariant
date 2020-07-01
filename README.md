# govariant

Generate variants/sum types/discriminated unions for Go

## Installation

```
go get github.com/mnxn/govariant
```

## Flags

| Flag       | Description                                                    |
| ---------- | -------------------------------------------------------------- |
| -nocheck   | disable generation of explicit interface implementation checks |
| -nounpack  | disable generation of unpack methods                           |
| -novisitor | disable generation of visitor struct and methods               |

## Usage

To annotate a interface so it is generated into a variant, use `@variant` followed by the variant name:

```go
// @variant Option
type _ interface {
	Some(value interface{})
	None()
}
```

Each interface method acts as a type constructor for the variant.

With all generation options disabled it will generate the following in `option_variant.go`:

```go
type Option interface{ isOption() }

type Some struct {
	value interface{}
}

type None struct {
}

func (Some) isOption() {}

func (None) isOption() {}
```

The underlying type of the generated types depends on the parameters of the type constructors.

| Source Type Constructor | Generated Type            |
| ----------------------- | ------------------------- |
| `A()`                   | `type A struct {}`        |
| `B(int)`                | `type B int`              |
| `C(x int)`              | `type C struct { x int }` |

Add a method that is required for all type constructors with `@method`:

```go
// @variant Option
type _ interface {
	Some(value interface{})
	None()

	String() string // @method
}
```

Which will add the method to the generated interface:

```go
type Option interface {
	String() string
	isOption()
}
```

With check generation enabled, the following will be added:

```go
var (
	_ Option = struct{ Some }{}
	_ Option = struct{ None }{}
)
```

If the type constructors don't implement the specified method, there will be an build error because of the checks:

```
.\option_variant.go:33:2: cannot use struct { Some } literal (type struct { Some }) as type Option in assignment:
        struct { Some } does not implement Option (missing String method)
.\option_variant.go:34:2: cannot use struct { None } literal (type struct { None }) as type Option in assignment:
        struct { None } does not implement Option (missing String method)
```

If unpack method generation is not disabled, `Unpack` methods are also generated:

```go
func (rcv Some) Unpack() interface{}   { return rcv.value }
func (rcv None) Unpack()               { return }
```

The `Unpack` method for an empty type isn't particularly useful, but it's provided for consistency.

They can be used from a type switch like so:

```go
switch option.(type) {
case Some:
	value := option.(Some).Unpack()
	fmt.Println("Value is", value)
case None:
	fmt.Println("No value")
}
```

If visitor generation is not disabled, a visitor struct and associated methods are generated:

```go
type Option interface {
	Visit(OptionVisitor)
	isOption()
}

type OptionVisitor struct {
	Some func(interface{})
	None func()
}

func (rcv Some) Visit(v OptionVisitor) { v.Some(rcv.value) }

func (rcv None) Visit(v OptionVisitor) { v.None() }
```

It can then be used in the following way:

```go
option.Visit(OptionVisitor{
	Some: func(value interface{}){
		fmt.Println("Value is", value)
	},
	None: func() {
		fmt.Println("No value")
	},
})
```

Neither the type switch nor visitor has exhaustivity checking.
External tools are necessary to ensure that each case is accounted for.

govariant can be used with go generate by adding the appropriate comment:

```
//go:generate govariant $GOFILE
```

Another full example is provided in the examples folder of this repository.
