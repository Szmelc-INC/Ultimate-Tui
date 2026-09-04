package widget

import tea "github.com/charmbracelet/bubbletea"

// keyToBytes translates a bubbletea key event into the byte sequence a terminal
// would send to a child process (xterm conventions). Returns nil for keys with
// no meaningful encoding.
func keyToBytes(k tea.KeyMsg) []byte {
	// Printable input.
	if k.Type == tea.KeyRunes {
		b := []byte(string(k.Runes))
		if k.Alt {
			return append([]byte{0x1b}, b...)
		}
		return b
	}

	switch k.Type {
	case tea.KeySpace:
		return []byte{' '}
	case tea.KeyBackspace:
		return []byte{0x7f}
	case tea.KeyDelete:
		return []byte("\x1b[3~")
	case tea.KeyInsert:
		return []byte("\x1b[2~")
	case tea.KeyUp:
		return []byte("\x1b[A")
	case tea.KeyDown:
		return []byte("\x1b[B")
	case tea.KeyRight:
		return []byte("\x1b[C")
	case tea.KeyLeft:
		return []byte("\x1b[D")
	case tea.KeyHome:
		return []byte("\x1b[H")
	case tea.KeyEnd:
		return []byte("\x1b[F")
	case tea.KeyPgUp:
		return []byte("\x1b[5~")
	case tea.KeyPgDown:
		return []byte("\x1b[6~")
	case tea.KeyShiftTab:
		return []byte("\x1b[Z")
	case tea.KeyF1:
		return []byte("\x1bOP")
	case tea.KeyF2:
		return []byte("\x1bOQ")
	case tea.KeyF3:
		return []byte("\x1bOR")
	case tea.KeyF4:
		return []byte("\x1bOS")
	case tea.KeyF5:
		return []byte("\x1b[15~")
	case tea.KeyF6:
		return []byte("\x1b[17~")
	case tea.KeyF7:
		return []byte("\x1b[18~")
	case tea.KeyF8:
		return []byte("\x1b[19~")
	case tea.KeyF9:
		return []byte("\x1b[20~")
	case tea.KeyF10:
		return []byte("\x1b[21~")
	case tea.KeyF11:
		return []byte("\x1b[23~")
	case tea.KeyF12:
		return []byte("\x1b[24~")
	}

	// Control range: bubbletea gives KeyCtrlA..KeyCtrlZ, KeyEnter (CR),
	// KeyTab (HT), KeyEscape (ESC) the KeyType value equal to their control
	// byte, so 0x00..0x1f maps straight through.
	if k.Type >= 0 && k.Type <= 0x1f {
		return []byte{byte(k.Type)}
	}
	return nil
}
