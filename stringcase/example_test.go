package stringcase_test

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/stringcase"
)

func Example() {
	fmt.Println(stringcase.Snake("OrderItem"))
	fmt.Println(stringcase.Camel("order_item"))
	fmt.Println(stringcase.Pascal("order_item"))
	fmt.Println(stringcase.Kebab("orderItem"))
	fmt.Println(stringcase.ScreamingSnake("orderItem"))
	// Output:
	// order_item
	// orderItem
	// OrderItem
	// order-item
	// ORDER_ITEM
}
