
package main

func TypeErrors() {
	var x int
	x = "string" // Type error

	var y string
	y = 123 // Type error

	var z interface{}
	z = func() {} // This is actually valid
}
