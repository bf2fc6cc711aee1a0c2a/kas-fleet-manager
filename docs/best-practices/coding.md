# Coding best practices
This document covers the best practices to be used when writing code for Kas Fleet Manager

## Method receivers: if unsure, use pointers
Always prefer
```go
func (x *myReceiverObject) myFancyMethod() {
	...
}
```
to
```go
func (x myReceiverObject) myFancyMethod() {
...
}
```

### Explanation
In `go`, method declaration is just syntactic sugar: the compiler will translate it to a method whose first parameter is the receiver.
So
```go
func (x myReceiverObject) myFancyMethod() {
	...
}
```
becomes:
```go
func myFancyMethod(x myReceiverObject) {
	...
}
```
That means that every time the method is invoked, the `myReceiverObject` is copied. This could even lead to weird issues. For example, the output of the code below
```go
package main

import "fmt"

type Contact struct {
	Name    string
	Surname string
}

func (c Contact) Update(name string, surname string) {
	c.Name = name
	c.Surname = surname
}

func main() {
	c := Contact{
		Name:    "Mickey",
		Surname: "Mouse",
	}
	fmt.Println("Before update", c)
	c.Update("Duffy", "Duck")
	fmt.Println("After update", c)
}
```
will be:
```
Before update {Mickey Mouse}
After update {Mickey Mouse}
```
That wouldn't happen using a pointer receiver:
```go
package main

import "fmt"

type Contact struct {
	Name    string
	Surname string
}

func (c *Contact) Update(name string, surname string) {
	c.Name = name
	c.Surname = surname
}

func main() {
	c := Contact{
		Name:    "Mickey",
		Surname: "Mouse",
	}
	fmt.Println("Before update", c)
	c.Update("Duffy", "Duck")
	fmt.Println("After update", c)
}
```
Output:
```
Before update {Mickey Mouse}
After update {Duffy Duck}
```
## Use the utility functions
If you find yourself writing a lot of repetitive code, there is a chance a utility code exists in `pkg/shared/utils`. If it doesn't, it can be worth reasoning whether it should be created.

### Getting first non-empty string
Avoid code like:
```go
myVar := val1
if myVar == "" {
	myVar = val2
}
if myVar == "" {
	myVar = val3
}
...
```
Instead, use this code:
```go
myVar, _ = arrays.FirstNonEmpty(val1, val2, val3)
```
or, if a default value is needed:
```go
myVar := FirstNonEmptyOrDefault("default value", val1, val2, val3)
```

For further details, read the [array utilities documentation](../utilities/arrays.md)

## Avoid exposing mutable internal state of objects
This issue is about blocking uncontrolled code from changing the internal state of an object.
An object should never return a reference or pointer to any internal state. Instead, a copy should be returned.

### Explanation
Suppose we have a `AddressBook` object that keeps an ordered list of `Contact`s:

```go
package main

import (
	"fmt"
	"sort"
)

type Contact struct {
	Name    string
	Surname string
}

type AddressBook struct {
	contacts []Contact
}

func (ab *AddressBook) AddContact(c *Contact) {
	ab.contacts = append(ab.contacts, *c)
	sort.Slice(ab.contacts, func(i, j int) bool { return ab.contacts[i].Surname < ab.contacts[j].Surname })
}

func (ab *AddressBook) GetAllContacts() []Contact {
	return ab.contacts
}

// PrintOrdered - The array is already ordered. Just loop it
func (ab *AddressBook) PrintOrdered() {
	for _, val := range ab.contacts {
		fmt.Println(val)
	}
}

// PrintReversedAddressBook - Prints all the contacts in the address book in reverse order
func PrintReversedAddressBook(ab AddressBook) {
	cl := ab.GetAllContacts()
	sort.Slice(cl, func(i, j int) bool { return cl[i].Surname > cl[j].Surname })
	for _, val := range cl {
		fmt.Println(val)
	}
}

func main() {
	ab := AddressBook{}
	c := Contact{
		Name:    "Mickey",
		Surname: "Mouse",
	}
	ab.AddContact(&c)
	c = Contact{
		Name:    "Duffy",
		Surname: "Duck",
	}
	ab.AddContact(&c)
	fmt.Println("Print ordered")
	ab.PrintOrdered()
	fmt.Println("Print reverse ordered")
	PrintReversedAddressBook(ab)
	fmt.Println("Print ordered")
	ab.PrintOrdered()
}
```
* The `PrintOrdered` method of the `AddressBook` just loops through the contacts without doing any check, since it knows that contacts can be added only through the `AddContact` method whose ensure the list remains ordered.
* The `PrintReversedAddressBook` method, takes a copy of the `AddressBook`, retrieve all the contacts calling the `GetAllContacts` method, reverses the array and prints it. Here is a bug: arrays are not deeply copied. Only the reference to the array is copied. That means if any of the object changes the array content, the change is visible to **all** the objects.

Running the previous code, the output is:
```
Print ordered
{Duffy Duck}
{Mickey Mouse}
Print reverse ordered
{Mickey Mouse}
{Duffy Duck}
Print ordered
{Mickey Mouse}
{Duffy Duck}
```
The `PrintReversedAddressBook` changed the order of the item into the internal `contacts` array of `AddressBook`
To fix that, the `GetAllContacts` method should return a *copy* of the `contacts` array:
```go
func (ab *AddressBook) GetAllContacts() []Contact {
	ret := make([]Contact, len(ab.contacts))
	copy(ret, ab.contacts)
	return ret
}
```
