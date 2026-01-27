
package main

func MissingImport() {
	// Using an unimported package
	x := make(chan int)
	close(x)
}
