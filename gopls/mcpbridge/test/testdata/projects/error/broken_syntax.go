
package main

import "fmt"

func BrokenSyntax() {
	// Missing closing brace
	x := func() {
		fmt.Println("hello")
	// }

	// Extra closing brace
}

func AnotherError() {
	var x int
	x = "string" // Type mismatch
}
