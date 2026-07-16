//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	cwUseDefault = 0x80000000

	wsOverlapped       = 0x00000000
	wsPopup            = 0x80000000
	wsChild            = 0x40000000
	wsMinimize         = 0x20000000
	wsVisible          = 0x10000000
	wsDisabled         = 0x08000000
	wsClipSiblings     = 0x04000000
	wsClipChildren     = 0x02000000
	wsCaption          = 0x00C00000
	wsBorder           = 0x00800000
	wsDlgFrame         = 0x00400000
	wsVScroll          = 0x00200000
	wsHScroll          = 0x00100000
	wsSysMenu          = 0x00080000
	wsThickFrame       = 0x00040000
	wsGroup            = 0x00020000
	wsTabStop          = 0x00010000
	wsMinimizeBox      = 0x00020000
	wsMaximizeBox      = 0x00010000
	wsOverlappedWindow = wsOverlapped | wsCaption | wsSysMenu | wsThickFrame | wsMinimizeBox | wsMaximizeBox

	wsExClientEdge = 0x00000200

	ssLeft   = 0x00000000
	ssCenter = 0x00000001

	bsPushButton    = 0x00000000
	bsDefPushButton = 0x00000001
	bsAutoCheckbox  = 0x00000003

	esLeft        = 0x00000000
	esMultiline   = 0x00000004
	esAutoVScroll = 0x00000040
	esReadOnly    = 0x00000800
	esWantReturn  = 0x00001000

	lbsNotify           = 0x00000001
	lbsNoIntegralHeight = 0x00000100
	lbsHasStrings       = 0x00000040

	swHide       = 0
	swShow       = 5
	swMinimize   = 6
	swRestore    = 9
	swShowNormal = 1

	wmCreate          = 0x0001
	wmDestroy         = 0x0002
	wmSize            = 0x0005
	wmPaint           = 0x000F
	wmClose           = 0x0010
	wmEraseBkgnd      = 0x0014
	wmCtlColorEdit    = 0x0133
	wmCtlColorListBox = 0x0134
	wmCtlColorBtn     = 0x0135
	wmCtlColorStatic  = 0x0138
	wmCommand         = 0x0111
	wmLButtonUp       = 0x0202
	wmLButtonDblClk   = 0x0203
	wmRButtonUp       = 0x0205
	wmApp             = 0x8000
	wmTray            = wmApp + 1
	wmSetFont         = 0x0030
	wmGetText         = 0x000D
	wmGetTextLength   = 0x000E
	wmSetText         = 0x000C

	bmGetCheck = 0x00F0
	bmSetCheck = 0x00F1
	bstChecked = 1

	lbAddString    = 0x0180
	lbResetContent = 0x0184
	lbSetCurSel    = 0x0186
	lbGetCurSel    = 0x0188
	lbErr          = -1
	lbnSelChange   = 1

	colorWindow = 5

	defaultGUIFont = 17
	nullBrush      = 5

	transparentBkMode = 1
	opaqueBkMode      = 2

	idcArrow = 32512

	ofnReadOnly      = 0x00000001
	ofnPathMustExist = 0x00000800
	ofnFileMustExist = 0x00001000
	ofnExplorer      = 0x00080000

	nimAdd             = 0x00000000
	nimDelete          = 0x00000002
	nimSetVersion      = 0x00000004
	nifMessage         = 0x00000001
	nifIcon            = 0x00000002
	nifTip             = 0x00000004
	notifyIconVersion4 = 4

	idiApplication = 32512

	mfString       = 0x00000000
	tpmRightButton = 0x00000002
	tpmReturnCmd   = 0x00000100

	biRGB        = 0
	dibRGBColors = 0
	srccopy      = 0x00CC0020

	hkeyCurrentUser = 0x80000001
	keyQueryValue   = 0x0001
	keySetValue     = 0x0002
	regSZ           = 1
	errorSuccess    = 0
)

type point struct {
	X int32
	Y int32
}

type msg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type paintStruct struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     rect
	FRestore    int32
	FIncUpdate  int32
	RGBReserved [32]byte
}

type bitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type bitmapInfo struct {
	BmiHeader bitmapInfoHeader
	BmiColors [1]uint32
}

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         guid
	HBalloonIcon     uintptr
}

type openFileName struct {
	LStructSize       uint32
	HwndOwner         uintptr
	HInstance         uintptr
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData         uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
	PvReserved        uintptr
	DwReserved        uint32
	FlagsEx           uint32
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procShowWindow          = user32.NewProc("ShowWindow")
	procUpdateWindow        = user32.NewProc("UpdateWindow")
	procMoveWindow          = user32.NewProc("MoveWindow")
	procEnableWindow        = user32.NewProc("EnableWindow")
	procInvalidateRect      = user32.NewProc("InvalidateRect")
	procSendMessageW        = user32.NewProc("SendMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procLoadCursorW         = user32.NewProc("LoadCursorW")
	procLoadIconW           = user32.NewProc("LoadIconW")
	procMessageBoxW         = user32.NewProc("MessageBoxW")
	procGetClientRect       = user32.NewProc("GetClientRect")
	procBeginPaint          = user32.NewProc("BeginPaint")
	procEndPaint            = user32.NewProc("EndPaint")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procDestroyMenu         = user32.NewProc("DestroyMenu")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	procGetStockObject   = gdi32.NewProc("GetStockObject")
	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procSetBkColor       = gdi32.NewProc("SetBkColor")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procSetTextColor     = gdi32.NewProc("SetTextColor")
	procStretchDIBits    = gdi32.NewProc("StretchDIBits")

	procShellExecuteW    = shell32.NewProc("ShellExecuteW")
	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
	procGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")

	procRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	procRegCreateKeyExW  = advapi32.NewProc("RegCreateKeyExW")
	procRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	procRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	procRegDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	procRegCloseKey      = advapi32.NewProc("RegCloseKey")
)

func mustUTF16(value string) *uint16 {
	return syscall.StringToUTF16Ptr(value)
}

func hInstance() uintptr {
	handle, _, _ := procGetModuleHandleW.Call(0)
	return handle
}

func loadArrowCursor() uintptr {
	cursor, _, _ := procLoadCursorW.Call(0, uintptr(idcArrow))
	return cursor
}

func loadApplicationIcon() uintptr {
	icon, _, _ := procLoadIconW.Call(0, uintptr(idiApplication))
	return icon
}

func registerWindowClass(className string, wndProc uintptr) error {
	name := mustUTF16(className)
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   wndProc,
		HInstance:     hInstance(),
		HCursor:       loadArrowCursor(),
		HbrBackground: uintptr(colorWindow + 1),
		LpszClassName: name,
	}
	result, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if result == 0 {
		return err
	}
	return nil
}

func createWindowEx(exStyle uint32, className, title string, style uint32, x, y, width, height int32, parent uintptr, id uintptr) uintptr {
	result, _, _ := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(mustUTF16(className))),
		uintptr(unsafe.Pointer(mustUTF16(title))),
		uintptr(style),
		uintptr(int32ToWindowArg(x)),
		uintptr(int32ToWindowArg(y)),
		uintptr(int32ToWindowArg(width)),
		uintptr(int32ToWindowArg(height)),
		parent,
		id,
		hInstance(),
		0,
	)
	return result
}

func int32ToWindowArg(value int32) uint32 {
	return uint32(value)
}

func defWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return result
}

func destroyWindow(hwnd uintptr) {
	procDestroyWindow.Call(hwnd)
}

func showWindow(hwnd uintptr, command int32) {
	procShowWindow.Call(hwnd, uintptr(command))
}

func updateWindow(hwnd uintptr) {
	procUpdateWindow.Call(hwnd)
}

func moveWindow(hwnd uintptr, x, y, width, height int32, repaint bool) {
	repaintValue := uintptr(0)
	if repaint {
		repaintValue = 1
	}
	procMoveWindow.Call(hwnd, uintptr(int32ToWindowArg(x)), uintptr(int32ToWindowArg(y)), uintptr(int32ToWindowArg(width)), uintptr(int32ToWindowArg(height)), repaintValue)
}

func invalidateWindow(hwnd uintptr) {
	procInvalidateRect.Call(hwnd, 0, 1)
}

func enableWindow(hwnd uintptr, enabled bool) {
	value := uintptr(0)
	if enabled {
		value = 1
	}
	procEnableWindow.Call(hwnd, value)
}

func sendMessage(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	result, _, _ := procSendMessageW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func postQuitMessage(exitCode int32) {
	procPostQuitMessage.Call(uintptr(exitCode))
}

func messageLoop() int {
	var message msg
	for {
		result, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) <= 0 {
			return int(message.WParam)
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func messageBox(owner uintptr, title, text string) {
	procMessageBoxW.Call(owner, uintptr(unsafe.Pointer(mustUTF16(text))), uintptr(unsafe.Pointer(mustUTF16(title))), 0)
}

func getClientRect(hwnd uintptr) rect {
	var rc rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	return rc
}

func beginPaint(hwnd uintptr) (uintptr, paintStruct) {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	return hdc, ps
}

func endPaint(hwnd uintptr, ps *paintStruct) {
	procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(ps)))
}

func stretchDIBits(hdc uintptr, destW, destH int32, srcX, srcY, srcW, srcH int32, pixels []byte, imageW, imageH int32) {
	if len(pixels) == 0 || destW <= 0 || destH <= 0 || srcW <= 0 || srcH <= 0 {
		return
	}
	info := bitmapInfo{
		BmiHeader: bitmapInfoHeader{
			BiSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			BiWidth:       imageW,
			BiHeight:      -imageH,
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: biRGB,
		},
	}
	procStretchDIBits.Call(
		hdc,
		0,
		0,
		uintptr(int32ToWindowArg(destW)),
		uintptr(int32ToWindowArg(destH)),
		uintptr(int32ToWindowArg(srcX)),
		uintptr(int32ToWindowArg(srcY)),
		uintptr(int32ToWindowArg(srcW)),
		uintptr(int32ToWindowArg(srcH)),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&info)),
		uintptr(dibRGBColors),
		uintptr(srccopy),
	)
}

func setForegroundWindow(hwnd uintptr) {
	procSetForegroundWindow.Call(hwnd)
}

func getCursorPos() point {
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return pt
}

func getStockFont() uintptr {
	result, _, _ := procGetStockObject.Call(uintptr(defaultGUIFont))
	return result
}

func getStockBrush(object int) uintptr {
	result, _, _ := procGetStockObject.Call(uintptr(object))
	return result
}

func createFont(height int32, weight int32, face string) uintptr {
	result, _, _ := procCreateFontW.Call(
		uintptr(int32ToWindowArg(height)),
		0,
		0,
		0,
		uintptr(int32ToWindowArg(weight)),
		0,
		0,
		0,
		1,
		0,
		0,
		5,
		0,
		uintptr(unsafe.Pointer(mustUTF16(face))),
	)
	return result
}

func createSolidBrush(color uint32) uintptr {
	result, _, _ := procCreateSolidBrush.Call(uintptr(color))
	return result
}

func setBkMode(hdc uintptr, mode int) {
	procSetBkMode.Call(hdc, uintptr(mode))
}

func setBkColor(hdc uintptr, color uint32) {
	procSetBkColor.Call(hdc, uintptr(color))
}

func setTextColor(hdc uintptr, color uint32) {
	procSetTextColor.Call(hdc, uintptr(color))
}

func rgb(red, green, blue byte) uint32 {
	return uint32(red) | uint32(green)<<8 | uint32(blue)<<16
}

func deleteObject(handle uintptr) {
	if handle != 0 {
		procDeleteObject.Call(handle)
	}
}

func setControlFont(hwnd uintptr, font uintptr) {
	if hwnd != 0 && font != 0 {
		sendMessage(hwnd, wmSetFont, font, 1)
	}
}

func setWindowText(hwnd uintptr, text string) {
	sendMessage(hwnd, wmSetText, 0, uintptr(unsafe.Pointer(mustUTF16(text))))
}

func getWindowText(hwnd uintptr) string {
	length := int(sendMessage(hwnd, wmGetTextLength, 0, 0))
	buffer := make([]uint16, length+1)
	if len(buffer) == 0 {
		return ""
	}
	sendMessage(hwnd, wmGetText, uintptr(len(buffer)), uintptr(unsafe.Pointer(&buffer[0])))
	return syscall.UTF16ToString(buffer)
}

func setCheckbox(hwnd uintptr, checked bool) {
	value := uintptr(0)
	if checked {
		value = bstChecked
	}
	sendMessage(hwnd, bmSetCheck, value, 0)
}

func checkboxChecked(hwnd uintptr) bool {
	return sendMessage(hwnd, bmGetCheck, 0, 0) == bstChecked
}

func resetListBox(hwnd uintptr) {
	sendMessage(hwnd, lbResetContent, 0, 0)
}

func addListBoxString(hwnd uintptr, text string) {
	sendMessage(hwnd, lbAddString, 0, uintptr(unsafe.Pointer(mustUTF16(text))))
}

func listBoxSelected(hwnd uintptr) int {
	result := int32(sendMessage(hwnd, lbGetCurSel, 0, 0))
	if result == lbErr {
		return -1
	}
	return int(result)
}

func selectListBox(hwnd uintptr, index int) {
	sendMessage(hwnd, lbSetCurSel, uintptr(index), 0)
}

func openFileDialog(owner uintptr, title, filter string) (string, bool) {
	buffer := make([]uint16, 1024)
	filterChars := syscall.StringToUTF16(filter)
	titlePtr := mustUTF16(title)
	ofn := openFileName{
		LStructSize: uint32(unsafe.Sizeof(openFileName{})),
		HwndOwner:   owner,
		LpstrFilter: &filterChars[0],
		LpstrFile:   &buffer[0],
		NMaxFile:    uint32(len(buffer)),
		LpstrTitle:  titlePtr,
		Flags:       ofnExplorer | ofnPathMustExist | ofnFileMustExist,
	}
	result, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if result == 0 {
		return "", false
	}
	return syscall.UTF16ToString(buffer), true
}

func shellOpen(owner uintptr, path string) error {
	result, _, _ := procShellExecuteW.Call(
		owner,
		uintptr(unsafe.Pointer(mustUTF16("open"))),
		uintptr(unsafe.Pointer(mustUTF16(path))),
		0,
		0,
		uintptr(swShowNormal),
	)
	if result <= 32 {
		return fmt.Errorf("无法打开：%s", path)
	}
	return nil
}

func addTrayIcon(hwnd uintptr, tip string) bool {
	data := newNotifyIconData(hwnd, tip)
	result, _, _ := procShellNotifyIconW.Call(uintptr(nimAdd), uintptr(unsafe.Pointer(&data)))
	if result == 0 {
		return false
	}
	data.UVersion = notifyIconVersion4
	procShellNotifyIconW.Call(uintptr(nimSetVersion), uintptr(unsafe.Pointer(&data)))
	return true
}

func deleteTrayIcon(hwnd uintptr) {
	data := notifyIconData{
		CbSize: uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:   hwnd,
		UID:    1,
	}
	procShellNotifyIconW.Call(uintptr(nimDelete), uintptr(unsafe.Pointer(&data)))
}

func newNotifyIconData(hwnd uintptr, tip string) notifyIconData {
	data := notifyIconData{
		CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:             hwnd,
		UID:              1,
		UFlags:           nifMessage | nifIcon | nifTip,
		UCallbackMessage: wmTray,
		HIcon:            loadApplicationIcon(),
	}
	tipChars := syscall.StringToUTF16(tip)
	copy(data.SzTip[:], tipChars)
	return data
}

func showTrayMenu(hwnd uintptr, showID, hideID, quitID int) int {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return 0
	}
	defer procDestroyMenu.Call(menu)

	appendMenu(menu, showID, "显示窗口")
	appendMenu(menu, hideID, "隐藏窗口")
	appendMenu(menu, quitID, "退出程序")

	pt := getCursorPos()
	setForegroundWindow(hwnd)
	command, _, _ := procTrackPopupMenu.Call(
		menu,
		uintptr(tpmRightButton|tpmReturnCmd),
		uintptr(int32ToWindowArg(pt.X)),
		uintptr(int32ToWindowArg(pt.Y)),
		0,
		hwnd,
		0,
	)
	return int(command)
}

func appendMenu(menu uintptr, id int, text string) {
	procAppendMenuW.Call(menu, uintptr(mfString), uintptr(id), uintptr(unsafe.Pointer(mustUTF16(text))))
}

func loword(value uintptr) uint16 {
	return uint16(value & 0xFFFF)
}

func hiword(value uintptr) uint16 {
	return uint16((value >> 16) & 0xFFFF)
}

func isAutostartEnabled() bool {
	key, ok := openRunKey(keyQueryValue)
	if !ok {
		return false
	}
	defer closeRegistryKey(key)

	name := mustUTF16(appName)
	var valueType uint32
	var size uint32
	result, _, _ := procRegQueryValueExW.Call(
		key,
		uintptr(unsafe.Pointer(name)),
		0,
		uintptr(unsafe.Pointer(&valueType)),
		0,
		uintptr(unsafe.Pointer(&size)),
	)
	return result == errorSuccess
}

func setAutostartEnabled(enabled bool) error {
	key, ok := createRunKey()
	if !ok {
		return fmt.Errorf("无法打开当前用户启动项注册表")
	}
	defer closeRegistryKey(key)

	name := mustUTF16(appName)
	if !enabled {
		procRegDeleteValueW.Call(key, uintptr(unsafe.Pointer(name)))
		return nil
	}

	commandChars := syscall.StringToUTF16(BuildAutostartCommand())
	result, _, _ := procRegSetValueExW.Call(
		key,
		uintptr(unsafe.Pointer(name)),
		0,
		uintptr(regSZ),
		uintptr(unsafe.Pointer(&commandChars[0])),
		uintptr(len(commandChars)*2),
	)
	if result != errorSuccess {
		return fmt.Errorf("写入自启动注册表失败，错误码：%d", result)
	}
	return nil
}

func openRunKey(access uint32) (uintptr, bool) {
	subkey := mustUTF16(`Software\Microsoft\Windows\CurrentVersion\Run`)
	var key uintptr
	result, _, _ := procRegOpenKeyExW.Call(
		uintptr(hkeyCurrentUser),
		uintptr(unsafe.Pointer(subkey)),
		0,
		uintptr(access),
		uintptr(unsafe.Pointer(&key)),
	)
	return key, result == errorSuccess
}

func createRunKey() (uintptr, bool) {
	subkey := mustUTF16(`Software\Microsoft\Windows\CurrentVersion\Run`)
	var key uintptr
	var disposition uint32
	result, _, _ := procRegCreateKeyExW.Call(
		uintptr(hkeyCurrentUser),
		uintptr(unsafe.Pointer(subkey)),
		0,
		0,
		0,
		uintptr(keySetValue),
		0,
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(&disposition)),
	)
	return key, result == errorSuccess
}

func closeRegistryKey(key uintptr) {
	if key != 0 {
		procRegCloseKey.Call(key)
	}
}
