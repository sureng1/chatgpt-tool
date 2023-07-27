package main

type RapedW struct {
	w     func(p []byte) (n int, err error)
	c     func() error
	flush func()
}

func (w *RapedW) Flush() {
	if w.flush == nil {
		return
	}
	w.flush()
}
func (w *RapedW) Write(p []byte) (n int, err error) {
	return w.w(p)
}
func (w *RapedW) Close() error {
	if w.c == nil {
		return nil
	}
	return w.c()
}

func NewWriter(
	w func(p []byte) (n int, err error),
	c func() error,
	flush func(),
) *RapedW {
	return &RapedW{
		w:     w,
		c:     c,
		flush: flush,
	}
}
