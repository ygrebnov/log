package log

import (
	"errors"
	"reflect"
	"testing"
)

type testCloser struct {
	close func() error
}

func (c *testCloser) Close() error {
	return c.close()
}

func TestCloserStack_Empty(t *testing.T) {
	c := newCloser()

	if err := c.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
}

func TestCloserStack_SingleResource(t *testing.T) {
	var calls int

	resource := &testCloser{
		close: func() error {
			calls++
			return nil
		},
	}

	c := newCloser(resource)

	if err := c.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}

	if calls != 1 {
		t.Errorf("Close() calls = %d, want 1", calls)
	}
}

func TestCloserStack_ReverseOrder(t *testing.T) {
	var order []string

	newResource := func(name string) *testCloser {
		return &testCloser{
			close: func() error {
				order = append(order, name)
				return nil
			},
		}
	}

	first := newResource("first")
	second := newResource("second")
	third := newResource("third")

	c := newCloser(first)
	c.Add(second)
	c.Add(third)

	if err := c.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}

	want := []string{
		"third",
		"second",
		"first",
	}

	if !reflect.DeepEqual(order, want) {
		t.Errorf(
			"close order = %v, want %v",
			order,
			want,
		)
	}
}

func TestCloserStack_ContinuesAfterError(t *testing.T) {
	errClose := errors.New("close failed")

	var order []string

	first := &testCloser{
		close: func() error {
			order = append(order, "first")
			return nil
		},
	}

	second := &testCloser{
		close: func() error {
			order = append(order, "second")
			return errClose
		},
	}

	third := &testCloser{
		close: func() error {
			order = append(order, "third")
			return nil
		},
	}

	c := newCloser(first, second, third)

	err := c.Close()

	if !errors.Is(err, errClose) {
		t.Errorf(
			"Close() error = %v, want errors.Is(err, errClose)",
			err,
		)
	}

	want := []string{
		"third",
		"second",
		"first",
	}

	if !reflect.DeepEqual(order, want) {
		t.Errorf(
			"close order = %v, want %v",
			order,
			want,
		)
	}
}

func TestCloserStack_MultipleErrors(t *testing.T) {
	errFirst := errors.New("first close failed")
	errSecond := errors.New("second close failed")

	first := &testCloser{
		close: func() error {
			return errFirst
		},
	}

	second := &testCloser{
		close: func() error {
			return errSecond
		},
	}

	third := &testCloser{
		close: func() error {
			return nil
		},
	}

	c := newCloser(first, second, third)

	err := c.Close()

	if err == nil {
		t.Fatal("Close() expected error, got nil")
	}

	if !errors.Is(err, errFirst) {
		t.Errorf(
			"Close() error = %v, want errors.Is(err, errFirst)",
			err,
		)
	}

	if !errors.Is(err, errSecond) {
		t.Errorf(
			"Close() error = %v, want errors.Is(err, errSecond)",
			err,
		)
	}
}
