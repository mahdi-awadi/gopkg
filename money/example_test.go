package money_test

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/money"
)

func ExampleNew() {
	price, _ := money.New(9.99, "USD")
	fmt.Println(price)
	fmt.Println(price.Minor())
	// Output:
	// 9.99 USD
	// 999
}

func ExampleMoney_Add() {
	subtotal, _ := money.New(10.00, "USD")
	tax, _ := money.New(0.85, "USD")
	total, _ := subtotal.Add(tax)
	fmt.Println(total)
	// Output: 10.85 USD
}

func ExampleFromMinor() {
	// KWD has 3 decimal places
	m, _ := money.FromMinor(12345, "KWD")
	fmt.Println(m)
	// Output: 12.345 KWD
}
