package main

import "fmt"

// Hello returns a greeting message
func Hello() string {
	return "hello world"
}

// Add returns the sum of two integers
func Add(a, b int) int {
	return a + b
}

// Person represents a person
type Person struct {
	Name string
	Age  int
}

// (Person) Greeting returns a greeting from the person
func (p *Person) Greeting() string {
	return fmt.Sprintf("Hello, my name is %s", p.Name)
}

func main() {
	fmt.Println(Hello())
	fmt.Println(Add(1, 2))
	p := Person{Name: "Alice", Age: 30}
	fmt.Println(p.Greeting())
}
