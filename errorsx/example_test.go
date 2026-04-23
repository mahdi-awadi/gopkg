package errorsx_test

import (
	"errors"
	"fmt"

	"github.com/mahdi-awadi/gopkg/errorsx"
)

func ExampleNew() {
	err := errorsx.New(errorsx.KindNotFound, "user 42 not found")
	fmt.Println(err.Error())
	fmt.Println(errorsx.HTTPStatus(err))
	fmt.Println(errors.Is(err, errorsx.ErrNotFound))
	// Output:
	// user 42 not found
	// 404
	// true
}

func ExampleWrap() {
	dbErr := errors.New("pg: no rows")
	err := errorsx.Wrap(errorsx.KindNotFound, dbErr, "loading user")
	fmt.Println(err.Error())
	fmt.Println(errorsx.HTTPStatus(err))
	// Output:
	// loading user: pg: no rows
	// 404
}
