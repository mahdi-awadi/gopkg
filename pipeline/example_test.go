package pipeline_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/pipeline"
)

type filterFunc[T any] struct {
	name string
	fn   func([]T) []T
}

func (f *filterFunc[T]) Name() string                    { return f.name }
func (f *filterFunc[T]) Criticality() pipeline.Criticality { return pipeline.Critical }
func (f *filterFunc[T]) Phase() pipeline.Phase            { return 0 }
func (f *filterFunc[T]) Apply(_ context.Context, xs []T, _ any) ([]T, error) {
	return f.fn(xs), nil
}

func Example() {
	keepPositive := &filterFunc[int]{
		name: "keep-positive",
		fn: func(xs []int) []int {
			out := xs[:0]
			for _, x := range xs {
				if x > 0 {
					out = append(out, x)
				}
			}
			return out
		},
	}
	sum := &filterFunc[int]{
		name: "sum",
		fn: func(xs []int) []int {
			total := 0
			for _, x := range xs {
				total += x
			}
			return []int{total}
		},
	}

	p := pipeline.New[int](nil, keepPositive, sum)
	out, logs, err := p.Run(context.Background(), []int{3, -1, 4, -5, 9}, nil)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	fmt.Println("result:", out[0])
	fmt.Println("steps:", len(logs))
	// Output:
	// result: 16
	// steps: 2
}
