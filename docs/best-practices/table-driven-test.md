# Table of Contents

<!-- toc -->

- [Table Driven Unit Tests](#table-driven-unit-tests)
  * [Example](#example)
    + [Without Table Driven Test](#without-table-driven-test)
    + [With Table Driven Test](#with-table-driven-test)

<!-- tocstop -->

# Table Driven Unit Tests

Instead of writing a test for each of the input values we want to test, with a Table Driven Test we can configure all the input and expected outcomes into a table.
Then we loop through all the table entries and run the necessary tests. This way, the test code is written once.

## Example

We want to test the following funcion:
```go
func Divide(a float32, b float32) (float32, error) {
    if b != 0 {
        return a / b, nil
    }
    return 0, fmt.Errorf("Division by zero")
}
```

### Without Table Driven Test
```go
func TestDividePositive(t *testing.T) {
    var a float32 = 9
    var b float32 = 4
    res, err := Divide(a,b)
    Expect(err).ToNot(HaveOccurred())
    Expect(res).ToEqual(a/b)
}

func TestDivideByZero(t * testing.T) {
    var a float32 = 9
    var b float32 = 0
    res, err := Divide(a,b)
    Expect(err).To(HaveOccurred())
}

...
```

### With Table Driven Test
```go
func TestDivide(t *testing.T) {
	type args struct {
        a float32
        b float32
    }
	tests := []struct {
        name    string
        args    args
        want    float32
        wantErr bool
    }{
        {
            name: "positive inputs",
            args: args{a:9, b:4},
            want: float32(9)/float32(4),
            wantErr: false,
        },
		{
            name: "divide by zero",
            args: args{a:9, b:0},
            want: float32(0),
            wantErr: true,
        },
		// ...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            res, err := Divide(tt.args.a, tt.args.b)
            Expect(err != nil).To(Equal(tt.wantErr))
            Expect(res).To(Equal(tt.want))
        })	
    }
}
```
