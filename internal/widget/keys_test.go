package widget

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyToBytes(t *testing.T) {
	cases := []struct {
		name string
		key  tea.KeyMsg
		want []byte
	}{
		{"letter", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, []byte("a")},
		{"utf8", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ł")}, []byte("ł")},
		{"alt-letter", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b"), Alt: true}, []byte{0x1b, 'b'}},
		{"space", tea.KeyMsg{Type: tea.KeySpace}, []byte{' '}},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}, []byte{'\r'}},
		{"tab", tea.KeyMsg{Type: tea.KeyTab}, []byte{'\t'}},
		{"backspace", tea.KeyMsg{Type: tea.KeyBackspace}, []byte{0x7f}},
		{"esc", tea.KeyMsg{Type: tea.KeyEscape}, []byte{0x1b}},
		{"ctrl-c", tea.KeyMsg{Type: tea.KeyCtrlC}, []byte{0x03}},
		{"ctrl-a", tea.KeyMsg{Type: tea.KeyCtrlA}, []byte{0x01}},
		{"ctrl-z", tea.KeyMsg{Type: tea.KeyCtrlZ}, []byte{0x1a}},
		{"up", tea.KeyMsg{Type: tea.KeyUp}, []byte("\x1b[A")},
		{"down", tea.KeyMsg{Type: tea.KeyDown}, []byte("\x1b[B")},
		{"right", tea.KeyMsg{Type: tea.KeyRight}, []byte("\x1b[C")},
		{"left", tea.KeyMsg{Type: tea.KeyLeft}, []byte("\x1b[D")},
		{"home", tea.KeyMsg{Type: tea.KeyHome}, []byte("\x1b[H")},
		{"pgup", tea.KeyMsg{Type: tea.KeyPgUp}, []byte("\x1b[5~")},
		{"delete", tea.KeyMsg{Type: tea.KeyDelete}, []byte("\x1b[3~")},
		{"f1", tea.KeyMsg{Type: tea.KeyF1}, []byte("\x1bOP")},
		{"f10", tea.KeyMsg{Type: tea.KeyF10}, []byte("\x1b[21~")},
		{"shift-tab", tea.KeyMsg{Type: tea.KeyShiftTab}, []byte("\x1b[Z")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := keyToBytes(tc.key); !bytes.Equal(got, tc.want) {
				t.Fatalf("keyToBytes(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
