package ui

import (
	"fmt"
	"io"
)

const (
	Reset = "\033[0m"
	Red   = "\033[31m"
	Green = "\033[32m]"
	Blue  = "\033[34m"
)

type Renderer struct {
	out io.Writer
}

func New(out io.Writer) *Renderer { return &Renderer{out: out} }

func (r *Renderer) Header(title string) {
	fmt.Fprintf(r.out, "\n== %s ==\n", title)
}

func (r *Renderer) Item(idx int, title, meta string) {
	fmt.Fprintf(r.out, "%2d) %s\n    %s\n", idx, title, meta)
}

func (r *Renderer) Info(msg string)  { fmt.Fprintf(r.out, "ℹ %s\n", msg) }
func (r *Renderer) Warn(msg string)  { fmt.Fprintf(r.out, "⚠ %s\n", msg) }
func (r *Renderer) Error(msg string) { fmt.Fprintf(r.out, "✗ %s\n", msg) }
