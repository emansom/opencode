//go:build linux && amd64

package complex

//go:generate stringer -type=Color

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// Color represents a color.
type Color int

const (
	Red   Color = iota
	Green
	Blue
)

// Inner is a nested struct.
type Inner struct {
	Value string `json:"value"`
}

// Outer contains nested structs.
type Outer struct {
	Name  string  `json:"name"`
	Inner Inner   `json:"inner"`
	Items []Inner `json:"items"`
}

// Processor processes items with complex signatures.
type Processor interface {
	Process(ctx context.Context, items []Inner) ([]Inner, error)
	io.Closer
}

// ProcessAll handles all items.
func ProcessAll(ctx context.Context, items []Inner, opts ...func(*Outer)) (*Outer, error) {
	if len(items) == 0 {
		return nil, errors.New("no items")
	}
	result := &Outer{Items: items}
	for _, opt := range opts {
		opt(result)
	}
	return result, nil
}

// Outer.Validate validates the outer struct.
func (o *Outer) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("name is required")
	}
	for i, item := range o.Items {
		if item.Value == "" {
			return fmt.Errorf("item %d: value is required", i)
		}
	}
	return nil
}

// Outer.String returns a string representation.
func (o *Outer) String() string {
	return fmt.Sprintf("Outer{Name: %s, Items: %d}", o.Name, len(o.Items))
}

func init() {
	_ = Red
	_ = Green
	_ = Blue
}
