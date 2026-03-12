package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// ClipboardItem represents a single item to be copied to the clipboard.
type ClipboardItem struct {
	Title string
	URL   string
}

var items []ClipboardItem
var mu sync.Mutex

// AppendClipboard adds items to the clipboard buffer.
func AppendClipboard(newItems []ClipboardItem) {
	mu.Lock()
	defer mu.Unlock()
	items = append(items, newItems...)
}

// FlushClipboard converts accumulated items to RTF and copies to clipboard.
func FlushClipboard() {
	if len(items) == 0 {
		return
	}
	var lines []string
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("<a href=\"%s\">%s</a>", item.URL, item.Title))
	}
	html := `<html><head><meta charset="utf-8"></head><body>` + strings.Join(lines, "<br>") + "</body></html>"
	// Convert HTML to RTF via textutil, then copy as rich text
	textutil := exec.Command("textutil", "-stdin", "-format", "html", "-convert", "rtf", "-stdout")
	textutil.Stdin = strings.NewReader(html)
	pbcopy := exec.Command("pbcopy", "-Prefer", "rtf")
	var err error
	pbcopy.Stdin, err = textutil.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31mClipboard copy failed: %v\033[0m\n", err)
		return
	}
	if err := pbcopy.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31mClipboard copy failed: %v\033[0m\n", err)
		return
	}
	if err := textutil.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31mClipboard copy failed: %v\033[0m\n", err)
		return
	}
	if err := pbcopy.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31mClipboard copy failed: %v\033[0m\n", err)
		return
	}
	fmt.Println("\n\033[2m📋 Copied to clipboard\033[0m")
}
