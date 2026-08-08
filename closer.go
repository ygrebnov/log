package log

import "errors"

type closerStack struct {
	stack []closer
}

type closer interface {
	Close() error
}

func newCloser(stack ...closer) *closerStack {
	return &closerStack{
		stack: stack,
	}
}

func (c *closerStack) Add(resource closer) {
	c.stack = append(c.stack, resource)
}

func (c *closerStack) Close() error {
	var closeErr error
	for i := len(c.stack) - 1; i >= 0; i-- {
		closeErr = errors.Join(
			closeErr,
			c.stack[i].Close(),
		)
	}
	return closeErr
}
