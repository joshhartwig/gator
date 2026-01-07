package ui

import (
	"fmt"
	"io"
	"text/tabwriter"
)

const (
	Reset = "\033[0m"
	Red   = "\033[31m"
	Green = "\033[32m"
	Blue  = "\033[34m"
)

type Renderer struct {
	out io.Writer
}

func New(out io.Writer) *Renderer { return &Renderer{out: out} }

func (r *Renderer) Header(title string) {
	fmt.Fprintf(r.out, "\n=== %s ===\n\n", title)
}

func (r *Renderer) Item(format string, args ...any) {
	fmt.Fprintf(r.out, format+"\n", args...)
}

func (r *Renderer) Column(format string, args ...any) {
	tw := tabwriter.NewWriter(r.out, 2, 2, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintf(tw, format, args...)
	tw.Flush()
}

func (r *Renderer) Info(msg string)  { fmt.Fprintf(r.out, "ℹ %s\n", msg) }
func (r *Renderer) Warn(msg string)  { fmt.Fprintf(r.out, "⚠ %s\n", msg) }
func (r *Renderer) Error(msg string) { fmt.Fprintf(r.out, "✗ %s\n", msg) }
