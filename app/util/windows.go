// +build windows

package util

import "syscall"

// HideConsole скрывает консоль приложения в Windows,
// Взято отсюда - https://github.com/syncthing/syncthing/blob/master/lib/osutil/hidden_windows.go
func HideConsole() {
	getConsoleWindow := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow")
	showWindow := syscall.NewLazyDLL("user32.dll").NewProc("ShowWindow")
	if getConsoleWindow.Find() == nil && showWindow.Find() == nil {
		hwnd, _, _ := getConsoleWindow.Call()
		if hwnd != 0 {
			showWindow.Call(hwnd, 0)
		}
	}
}
