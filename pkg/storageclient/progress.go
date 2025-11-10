package storageclient

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	progressBarWidth     = 32
	progressRenderPeriod = 120 * time.Millisecond
)

// progressBar рисует ASCII-индикатор выполнения для потоков данных.
type progressBar struct {
	prefix        string
	total         int64
	current       int64
	lastRender    time.Time
	lastLineWidth int
	finished      bool
	mu            sync.Mutex
}

func newProgressBar(prefix string, total int64) *progressBar {
	return &progressBar{
		prefix: prefix,
		total:  total,
	}
}

func (p *progressBar) AddBytes(n int64) {
	if p == nil || n <= 0 {
		return
	}
	p.mu.Lock()
	if p.finished {
		p.mu.Unlock()
		return
	}
	p.current += n
	p.mu.Unlock()
	p.render(false, "")
}

func (p *progressBar) render(force bool, suffix string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.finished && !force {
		p.mu.Unlock()
		return
	}
	now := time.Now()
	if !force && now.Sub(p.lastRender) < progressRenderPeriod {
		p.mu.Unlock()
		return
	}

	line := p.lineLocked()
	prevWidth := p.lastLineWidth
	p.lastLineWidth = len(line) + len(suffix)
	p.lastRender = now
	p.mu.Unlock()

	padding := ""
	if prevWidth > len(line)+len(suffix) {
		padding = strings.Repeat(" ", prevWidth-len(line)-len(suffix))
	}
	fmt.Fprintf(os.Stdout, "\r%s%s%s", line, suffix, padding)
}

func (p *progressBar) lineLocked() string {
	var builder strings.Builder
	builder.Grow(len(p.prefix) + 64)
	builder.WriteString(p.prefix)
	builder.WriteByte(' ')

	if p.total > 0 {
		ratio := float64(0)
		if p.total > 0 {
			ratio = float64(p.current) / float64(p.total)
		}
		if ratio > 1 {
			ratio = 1
		}
		filled := int(ratio*float64(progressBarWidth) + 0.5)
		if filled > progressBarWidth {
			filled = progressBarWidth
		}
		builder.WriteByte('[')
		builder.WriteString(strings.Repeat("=", filled))
		builder.WriteString(strings.Repeat(" ", progressBarWidth-filled))
		builder.WriteString("] ")
		builder.WriteString(fmt.Sprintf("%3d%% ", int(ratio*100+0.5)))
		builder.WriteString(humanBytes(p.current))
		builder.WriteByte('/')
		builder.WriteString(humanBytes(p.total))
	} else {
		builder.WriteString(humanBytes(p.current))
		builder.WriteString(" transferred")
	}

	return builder.String()
}

func (p *progressBar) Finish() {
	p.complete(true, nil)
}

func (p *progressBar) Fail(err error) {
	p.complete(false, err)
}

func (p *progressBar) complete(success bool, err error) {
	if p == nil {
		return
	}

	p.mu.Lock()
	if p.finished {
		p.mu.Unlock()
		return
	}
	p.finished = true
	line := p.lineLocked()
	prevWidth := p.lastLineWidth
	p.lastLineWidth = len(line)
	p.mu.Unlock()

	suffix := " ✓"
	if !success {
		if err != nil {
			suffix = fmt.Sprintf(" ✗ %v", err)
		} else {
			suffix = " ✗"
		}
	}

	padding := ""
	if prevWidth > len(line)+len(suffix) {
		padding = strings.Repeat(" ", prevWidth-len(line)-len(suffix))
	}

	fmt.Fprintf(os.Stdout, "\r%s%s%s\n", line, suffix, padding)
}

type progressWriter struct {
	bar *progressBar
}

func (w progressWriter) Write(p []byte) (int, error) {
	if len(p) > 0 && w.bar != nil {
		w.bar.AddBytes(int64(len(p)))
	}
	return len(p), nil
}

type progressReadCloser struct {
	inner io.ReadCloser
	bar   *progressBar
	done  bool
}

func newProgressReadCloser(inner io.ReadCloser, bar *progressBar) io.ReadCloser {
	if bar == nil || inner == nil {
		return inner
	}

	return &progressReadCloser{
		inner: inner,
		bar:   bar,
	}
}

func (p *progressReadCloser) Read(b []byte) (int, error) {
	n, err := p.inner.Read(b)
	if n > 0 && p.bar != nil {
		p.bar.AddBytes(int64(n))
	}
	if err != nil {
		p.finish(err)
	}
	return n, err
}

func (p *progressReadCloser) Close() error {
	err := p.inner.Close()
	p.finish(err)
	return err
}

func (p *progressReadCloser) finish(err error) {
	if p == nil || p.done {
		return
	}
	p.done = true
	if err != nil && err != io.EOF {
		p.bar.Fail(err)
		return
	}
	p.bar.Finish()
}

func humanBytes(v int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	value := float64(v)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", v, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}
