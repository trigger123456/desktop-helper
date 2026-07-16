//go:build windows

package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const (
	idOpenData = 1001
	idHide     = 1002
	idQuit     = 1003

	idNavStudy    = 1101
	idNavWords    = 1102
	idNavMemo     = 1103
	idNavPreview  = 1104
	idNavSettings = 1105
	idNavHelp     = 1106

	idReloadStudy = 1201
	idMarkUnknown = 1202
	idMarkKnown   = 1203
	idPrevious    = 1204
	idNext        = 1205

	idSearch      = 1301
	idWordsList   = 1302
	idReloadWords = 1303
	idAddWord     = 1304
	idChooseCSV   = 1305
	idImportCSV   = 1306

	idSaveMemo = 1401

	idOpenBackground   = 1501
	idChooseBackground = 1502
	idResetBackground  = 1503

	idSaveSettings = 1601

	idOpenReadme = 1651

	idTrayShow = 1701
	idTrayHide = 1702
	idTrayQuit = 1703

	enChange = 0x0300
)

type uiApp struct {
	paths    Paths
	settings Settings
	progress Progress
	words    []Word

	todayWords    []Word
	filteredWords []Word
	currentIndex  int
	activePage    string
	isQuitting    bool
	trayIconAdded bool

	backgroundPath   string
	backgroundPixels []byte
	backgroundWidth  int32
	backgroundHeight int32
	backgroundErr    string

	hwnd        uintptr
	font        uintptr
	titleFont   uintptr
	wordFont    uintptr
	monoFont    uintptr
	pageCtrls   map[string][]uintptr
	inputBrush  uintptr
	buttonBrush uintptr
	inputColor  uint32
	buttonColor uint32
	textColor   uint32

	titleLabel    uintptr
	subtitleLabel uintptr
	openDataBtn   uintptr
	hideBtn       uintptr
	quitBtn       uintptr
	statusLabel   uintptr

	navStudy    uintptr
	navWords    uintptr
	navMemo     uintptr
	navPreview  uintptr
	navSettings uintptr
	navHelp     uintptr

	studyTitle    uintptr
	studyCounter  uintptr
	reloadStudy   uintptr
	metricTotal   uintptr
	metricKnown   uintptr
	metricUnknown uintptr
	metricToday   uintptr
	wordLabel     uintptr
	meaningLabel  uintptr
	exampleLabel  uintptr
	tagLabel      uintptr
	unknownBtn    uintptr
	knownBtn      uintptr
	prevBtn       uintptr
	nextBtn       uintptr

	searchEdit     uintptr
	wordsList      uintptr
	wordDetail     uintptr
	reloadWordsBtn uintptr
	addWordEdit    uintptr
	addMeaningEdit uintptr
	addTagEdit     uintptr
	addExampleEdit uintptr
	addWordBtn     uintptr
	importPathEdit uintptr
	chooseCSVBtn   uintptr
	importCSVBtn   uintptr

	memoEdit    uintptr
	saveMemoBtn uintptr
	memoHint    uintptr

	previewTitle uintptr
	previewText  uintptr
	openBgBtn    uintptr
	chooseBgBtn  uintptr
	resetBgBtn   uintptr

	dailyGoalEdit   uintptr
	startHiddenChk  uintptr
	autostartChk    uintptr
	bgEnabledChk    uintptr
	bgVisibleChk    uintptr
	opacityEdit     uintptr
	bgPathEdit      uintptr
	saveSettingsBtn uintptr
	settingsInfo    uintptr

	helpTitle     uintptr
	helpText      uintptr
	openReadmeBtn uintptr
}

var appInstance *uiApp

func main() {
	app := &uiApp{
		paths:     AppPaths(),
		pageCtrls: map[string][]uintptr{},
	}

	if err := EnsureDataFiles(app.paths); err != nil {
		fmt.Println("初始化数据目录失败：", err)
		return
	}
	app.loadState()
	appInstance = app

	className := "DesktopHelperGoWindow"
	if err := registerWindowClass(className, syscall.NewCallback(windowProc)); err != nil {
		fmt.Println("注册窗口失败：", err)
		return
	}

	hwnd := createWindowEx(
		0,
		className,
		"Desktop Helper "+appVersion,
		wsOverlappedWindow,
		100,
		80,
		1040,
		680,
		0,
		0,
	)
	if hwnd == 0 {
		fmt.Println("创建窗口失败")
		return
	}

	if app.settings.StartHidden || hasArg("--hidden") {
		showWindow(hwnd, swHide)
	} else {
		showWindow(hwnd, swShow)
	}
	updateWindow(hwnd)
	messageLoop()
}

func hasArg(value string) bool {
	for _, arg := range os.Args[1:] {
		if arg == value {
			return true
		}
	}
	return false
}

func windowProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	app := appInstance
	switch message {
	case wmCreate:
		app.hwnd = hwnd
		app.createControls()
		app.syncSettingsToControls()
		app.refreshAll()
		setWindowText(app.memoEdit, ReadMemo(app.paths))
		app.showPage("study")
		app.trayIconAdded = addTrayIcon(hwnd, "Desktop Helper "+appVersion)
		app.layout()
		if app.trayIconAdded {
			app.setStatus("Go 版界面已启动。关闭窗口会隐藏到系统托盘。")
		} else {
			app.setStatus("Go 版界面已启动。托盘图标未启用，关闭窗口会最小化。")
		}
		return 0
	case wmSize:
		if app != nil {
			app.layout()
		}
		return 0
	case wmPaint:
		if app != nil && app.paintBackground() {
			return 0
		}
	case wmEraseBkgnd:
		if app != nil && app.hasBackground() {
			return 1
		}
	case wmCtlColorStatic, wmCtlColorBtn, wmCtlColorEdit, wmCtlColorListBox:
		if app != nil {
			return app.handleControlColor(message, wParam)
		}
	case wmCommand:
		if app != nil {
			app.handleCommand(int(loword(wParam)), hiword(wParam))
		}
		return 0
	case wmTray:
		if app != nil {
			app.handleTrayEvent(lParam)
		}
		return 0
	case wmClose:
		if app != nil && !app.isQuitting {
			app.hideWindow()
			return 0
		}
	case wmDestroy:
		if app != nil {
			app.cleanup()
		}
		postQuitMessage(0)
		return 0
	}
	return defWindowProc(hwnd, message, wParam, lParam)
}

func (a *uiApp) loadState() {
	a.settings = LoadSettings(a.paths)
	a.progress = LoadProgress(a.paths)
	a.words = LoadWords(a.paths)
	a.todayWords = BuildTodayWords(a.words, a.progress, a.settings.DailyGoal)
}

func (a *uiApp) createControls() {
	a.font = getStockFont()
	a.titleFont = createFont(-22, 700, "Microsoft YaHei UI")
	a.wordFont = createFont(-42, 700, "Segoe UI")
	a.monoFont = createFont(-16, 400, "Consolas")
	a.textColor = rgb(23, 32, 51)
	a.inputColor = rgb(248, 250, 252)
	a.buttonColor = rgb(232, 238, 247)
	a.inputBrush = createSolidBrush(a.inputColor)
	a.buttonBrush = createSolidBrush(a.buttonColor)

	a.titleLabel = a.control("", "STATIC", "Desktop Helper "+appVersion, wsChild|wsVisible|ssLeft, 0, 0, a.titleFont)
	a.subtitleLabel = a.control("", "STATIC", "轻量单词学习与备忘录工具，Go 原生 Windows 界面。", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.openDataBtn = a.control("", "BUTTON", "数据目录", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idOpenData, a.font)
	a.hideBtn = a.control("", "BUTTON", "最小化", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idHide, a.font)
	a.quitBtn = a.control("", "BUTTON", "退出", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idQuit, a.font)
	a.statusLabel = a.control("", "STATIC", "", wsChild|wsVisible|ssLeft, 0, 0, a.font)

	a.navStudy = a.control("", "BUTTON", "今日学习", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idNavStudy, a.font)
	a.navWords = a.control("", "BUTTON", "单词列表", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idNavWords, a.font)
	a.navMemo = a.control("", "BUTTON", "备忘录", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idNavMemo, a.font)
	a.navPreview = a.control("", "BUTTON", "背景预览", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idNavPreview, a.font)
	a.navSettings = a.control("", "BUTTON", "设置", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idNavSettings, a.font)
	a.navHelp = a.control("", "BUTTON", "使用说明", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idNavHelp, a.font)

	a.createStudyPage()
	a.createWordsPage()
	a.createMemoPage()
	a.createPreviewPage()
	a.createSettingsPage()
	a.createHelpPage()
}

func (a *uiApp) createStudyPage() {
	a.studyTitle = a.control("study", "STATIC", "今日学习", wsChild|wsVisible|ssLeft, 0, 0, a.titleFont)
	a.studyCounter = a.control("study", "STATIC", "", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.reloadStudy = a.control("study", "BUTTON", "重新加载", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idReloadStudy, a.font)
	a.metricTotal = a.control("study", "STATIC", "", wsChild|wsVisible|ssLeft, wsExClientEdge, 0, a.font)
	a.metricKnown = a.control("study", "STATIC", "", wsChild|wsVisible|ssLeft, wsExClientEdge, 0, a.font)
	a.metricUnknown = a.control("study", "STATIC", "", wsChild|wsVisible|ssLeft, wsExClientEdge, 0, a.font)
	a.metricToday = a.control("study", "STATIC", "", wsChild|wsVisible|ssLeft, wsExClientEdge, 0, a.font)
	a.wordLabel = a.control("study", "STATIC", "", wsChild|wsVisible|ssLeft, 0, 0, a.wordFont)
	a.meaningLabel = a.control("study", "STATIC", "", wsChild|wsVisible|ssLeft, 0, 0, a.titleFont)
	a.exampleLabel = a.control("study", "STATIC", "", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.tagLabel = a.control("study", "STATIC", "", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.unknownBtn = a.control("study", "BUTTON", "不认识", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idMarkUnknown, a.font)
	a.knownBtn = a.control("study", "BUTTON", "认识", wsChild|wsVisible|wsTabStop|bsDefPushButton, 0, idMarkKnown, a.font)
	a.prevBtn = a.control("study", "BUTTON", "上一个", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idPrevious, a.font)
	a.nextBtn = a.control("study", "BUTTON", "下一个", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idNext, a.font)
}

func (a *uiApp) createWordsPage() {
	a.control("words", "STATIC", "搜索", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.searchEdit = a.control("words", "EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esLeft, wsExClientEdge, idSearch, a.font)
	a.reloadWordsBtn = a.control("words", "BUTTON", "重新加载", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idReloadWords, a.font)
	a.wordsList = a.control("words", "LISTBOX", "", wsChild|wsVisible|wsTabStop|wsVScroll|lbsNotify|lbsHasStrings|lbsNoIntegralHeight, wsExClientEdge, idWordsList, a.monoFont)
	a.wordDetail = a.control("words", "EDIT", "", wsChild|wsVisible|wsVScroll|esMultiline|esAutoVScroll|esReadOnly, wsExClientEdge, 0, a.font)

	a.control("words", "STATIC", "手动添加", wsChild|wsVisible|ssLeft, 0, 0, a.titleFont)
	a.control("words", "STATIC", "单词", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.addWordEdit = a.control("words", "EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esLeft, wsExClientEdge, 0, a.font)
	a.control("words", "STATIC", "释义", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.addMeaningEdit = a.control("words", "EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esLeft, wsExClientEdge, 0, a.font)
	a.control("words", "STATIC", "标签", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.addTagEdit = a.control("words", "EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esLeft, wsExClientEdge, 0, a.font)
	a.control("words", "STATIC", "例句", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.addExampleEdit = a.control("words", "EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esMultiline|esAutoVScroll|esWantReturn, wsExClientEdge, 0, a.font)
	a.addWordBtn = a.control("words", "BUTTON", "保存单词", wsChild|wsVisible|wsTabStop|bsDefPushButton, 0, idAddWord, a.font)

	a.control("words", "STATIC", "导入 CSV", wsChild|wsVisible|ssLeft, 0, 0, a.titleFont)
	a.importPathEdit = a.control("words", "EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esLeft, wsExClientEdge, 0, a.font)
	a.chooseCSVBtn = a.control("words", "BUTTON", "选择", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idChooseCSV, a.font)
	a.importCSVBtn = a.control("words", "BUTTON", "导入", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idImportCSV, a.font)
}

func (a *uiApp) createMemoPage() {
	a.memoEdit = a.control("memo", "EDIT", "", wsChild|wsVisible|wsTabStop|wsVScroll|esMultiline|esAutoVScroll|esWantReturn, wsExClientEdge, 0, a.font)
	a.saveMemoBtn = a.control("memo", "BUTTON", "保存备忘录", wsChild|wsVisible|wsTabStop|bsDefPushButton, 0, idSaveMemo, a.font)
	a.memoHint = a.control("memo", "STATIC", a.paths.MemoPath, wsChild|wsVisible|ssLeft, 0, 0, a.font)
}

func (a *uiApp) createPreviewPage() {
	a.previewTitle = a.control("preview", "STATIC", "背景预览", wsChild|wsVisible|ssLeft, 0, 0, a.titleFont)
	a.previewText = a.control("preview", "EDIT", "", wsChild|wsVisible|esMultiline|esAutoVScroll|esReadOnly, wsExClientEdge, 0, a.font)
	a.openBgBtn = a.control("preview", "BUTTON", "打开图片", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idOpenBackground, a.font)
	a.chooseBgBtn = a.control("preview", "BUTTON", "更换背景", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idChooseBackground, a.font)
	a.resetBgBtn = a.control("preview", "BUTTON", "恢复默认", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idResetBackground, a.font)
}

func (a *uiApp) createSettingsPage() {
	a.control("settings", "STATIC", "每日学习数量", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.dailyGoalEdit = a.control("settings", "EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esLeft, wsExClientEdge, 0, a.font)
	a.startHiddenChk = a.control("settings", "BUTTON", "启动后自动最小化", wsChild|wsVisible|wsTabStop|bsAutoCheckbox, 0, 0, a.font)
	a.autostartChk = a.control("settings", "BUTTON", "Windows 开机自启动", wsChild|wsVisible|wsTabStop|bsAutoCheckbox, 0, 0, a.font)
	a.bgEnabledChk = a.control("settings", "BUTTON", "启用背景图", wsChild|wsVisible|wsTabStop|bsAutoCheckbox, 0, 0, a.font)
	a.bgVisibleChk = a.control("settings", "BUTTON", "背景可见模式", wsChild|wsVisible|wsTabStop|bsAutoCheckbox, 0, 0, a.font)
	a.control("settings", "STATIC", "界面淡化程度 70-100", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.opacityEdit = a.control("settings", "EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esLeft, wsExClientEdge, 0, a.font)
	a.control("settings", "STATIC", "背景图片路径", wsChild|wsVisible|ssLeft, 0, 0, a.font)
	a.bgPathEdit = a.control("settings", "EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esLeft, wsExClientEdge, 0, a.font)
	a.saveSettingsBtn = a.control("settings", "BUTTON", "保存设置", wsChild|wsVisible|wsTabStop|bsDefPushButton, 0, idSaveSettings, a.font)
	a.settingsInfo = a.control("settings", "EDIT", "", wsChild|wsVisible|wsVScroll|esMultiline|esAutoVScroll|esReadOnly, wsExClientEdge, 0, a.font)
}

func (a *uiApp) createHelpPage() {
	a.helpTitle = a.control("help", "STATIC", "使用说明", wsChild|wsVisible|ssLeft, 0, 0, a.titleFont)
	a.helpText = a.control("help", "EDIT", "", wsChild|wsVisible|wsVScroll|esMultiline|esAutoVScroll|esReadOnly, wsExClientEdge, 0, a.font)
	a.openReadmeBtn = a.control("help", "BUTTON", "打开 README", wsChild|wsVisible|wsTabStop|bsPushButton, 0, idOpenReadme, a.font)
	setWindowText(a.helpText, a.helpContent())
}

func (a *uiApp) control(page, className, text string, style uint32, exStyle uint32, id int, font uintptr) uintptr {
	h := createWindowEx(exStyle, className, text, style, 0, 0, 10, 10, a.hwnd, uintptr(id))
	setControlFont(h, font)
	if page != "" {
		a.pageCtrls[page] = append(a.pageCtrls[page], h)
	}
	return h
}

func (a *uiApp) layout() {
	if a.hwnd == 0 {
		return
	}
	rc := getClientRect(a.hwnd)
	width := rc.Right - rc.Left
	height := rc.Bottom - rc.Top
	if width < 760 || height < 520 {
		width = 760
		height = 520
	}

	margin := int32(18)
	if a.settings.BackgroundEnabled && a.settings.BackgroundVisibleMode && a.hasBackground() {
		margin = 34
	}
	headerH := int32(64)
	statusH := int32(26)
	navW := int32(142)
	contentX := margin + navW + 14
	contentY := margin + headerH + 14
	contentW := width - contentX - margin
	contentH := height - contentY - statusH - margin - 8

	moveWindow(a.titleLabel, margin, 14, width-420, 28, true)
	moveWindow(a.subtitleLabel, margin, 42, width-420, 24, true)
	moveWindow(a.openDataBtn, width-300, 24, 88, 32, true)
	moveWindow(a.hideBtn, width-204, 24, 88, 32, true)
	moveWindow(a.quitBtn, width-108, 24, 88, 32, true)
	moveWindow(a.statusLabel, margin, height-statusH-margin/2, width-margin*2, statusH, true)

	navY := contentY
	for _, item := range []uintptr{a.navStudy, a.navWords, a.navMemo, a.navPreview, a.navSettings, a.navHelp} {
		moveWindow(item, margin, navY, navW, 38, true)
		navY += 46
	}

	switch a.activePage {
	case "words":
		a.layoutWords(contentX, contentY, contentW, contentH)
	case "memo":
		a.layoutMemo(contentX, contentY, contentW, contentH)
	case "preview":
		a.layoutPreview(contentX, contentY, contentW, contentH)
	case "settings":
		a.layoutSettings(contentX, contentY, contentW, contentH)
	case "help":
		a.layoutHelp(contentX, contentY, contentW, contentH)
	default:
		a.layoutStudy(contentX, contentY, contentW, contentH)
	}
}

func (a *uiApp) layoutStudy(x, y, w, h int32) {
	moveWindow(a.studyTitle, x, y, w-130, 30, true)
	moveWindow(a.reloadStudy, x+w-112, y, 112, 32, true)
	moveWindow(a.studyCounter, x, y+34, w, 24, true)

	cardW := (w - 24) / 4
	metricsY := y + 72
	moveWindow(a.metricTotal, x, metricsY, cardW, 58, true)
	moveWindow(a.metricKnown, x+cardW+8, metricsY, cardW, 58, true)
	moveWindow(a.metricUnknown, x+(cardW+8)*2, metricsY, cardW, 58, true)
	moveWindow(a.metricToday, x+(cardW+8)*3, metricsY, cardW, 58, true)

	wordY := metricsY + 88
	moveWindow(a.wordLabel, x, wordY, w, 58, true)
	moveWindow(a.meaningLabel, x, wordY+70, w, 36, true)
	moveWindow(a.exampleLabel, x, wordY+118, w, 58, true)
	moveWindow(a.tagLabel, x, wordY+186, w, 30, true)

	actionY := y + h - 44
	moveWindow(a.unknownBtn, x, actionY, 94, 36, true)
	moveWindow(a.knownBtn, x+104, actionY, 94, 36, true)
	moveWindow(a.prevBtn, x+216, actionY, 94, 36, true)
	moveWindow(a.nextBtn, x+320, actionY, 94, 36, true)
}

func (a *uiApp) layoutWords(x, y, w, h int32) {
	leftW := w * 58 / 100
	rightX := x + leftW + 16
	rightW := w - leftW - 16

	controls := a.pageCtrls["words"]
	if len(controls) >= 16 {
		moveWindow(controls[0], x, y+6, 44, 24, true)
	}
	moveWindow(a.searchEdit, x+48, y, leftW-170, 32, true)
	moveWindow(a.reloadWordsBtn, x+leftW-112, y, 112, 32, true)
	moveWindow(a.wordsList, x, y+42, leftW, h*58/100, true)
	moveWindow(a.wordDetail, x, y+54+h*58/100, leftW, h-(h*58/100)-54, true)

	if len(controls) >= 17 {
		moveWindow(controls[5], rightX, y, rightW, 30, true)
		moveWindow(controls[6], rightX, y+48, 56, 24, true)
		moveWindow(a.addWordEdit, rightX+64, y+42, rightW-64, 32, true)
		moveWindow(controls[8], rightX, y+88, 56, 24, true)
		moveWindow(a.addMeaningEdit, rightX+64, y+82, rightW-64, 32, true)
		moveWindow(controls[10], rightX, y+128, 56, 24, true)
		moveWindow(a.addTagEdit, rightX+64, y+122, rightW-64, 32, true)
		moveWindow(controls[12], rightX, y+168, 56, 24, true)
		moveWindow(a.addExampleEdit, rightX+64, y+162, rightW-64, 78, true)
		moveWindow(a.addWordBtn, rightX+rightW-110, y+252, 110, 34, true)

		importY := y + 316
		moveWindow(controls[15], rightX, importY, rightW, 30, true)
		moveWindow(a.importPathEdit, rightX, importY+42, rightW-150, 32, true)
		moveWindow(a.chooseCSVBtn, rightX+rightW-142, importY+42, 66, 32, true)
		moveWindow(a.importCSVBtn, rightX+rightW-70, importY+42, 70, 32, true)
	}
}

func (a *uiApp) layoutMemo(x, y, w, h int32) {
	moveWindow(a.memoEdit, x, y, w, h-52, true)
	moveWindow(a.saveMemoBtn, x, y+h-40, 120, 34, true)
	moveWindow(a.memoHint, x+132, y+h-34, w-132, 24, true)
}

func (a *uiApp) layoutPreview(x, y, w, h int32) {
	moveWindow(a.previewTitle, x, y, w, 34, true)
	moveWindow(a.previewText, x, y+46, w, h-104, true)
	moveWindow(a.openBgBtn, x, y+h-42, 96, 34, true)
	moveWindow(a.chooseBgBtn, x+106, y+h-42, 102, 34, true)
	moveWindow(a.resetBgBtn, x+218, y+h-42, 102, 34, true)
}

func (a *uiApp) layoutSettings(x, y, w, h int32) {
	labelW := int32(170)
	rowH := int32(40)
	infoX := x + w/2 + 16
	infoW := w/2 - 16
	leftW := w/2 - 20

	controls := a.pageCtrls["settings"]
	if len(controls) >= 11 {
		moveWindow(controls[0], x, y+6, labelW, 24, true)
		moveWindow(a.dailyGoalEdit, x+labelW, y, 90, 32, true)
		moveWindow(a.startHiddenChk, x, y+rowH, leftW, 28, true)
		moveWindow(a.autostartChk, x, y+rowH*2, leftW, 28, true)
		moveWindow(a.bgEnabledChk, x, y+rowH*3, leftW, 28, true)
		moveWindow(a.bgVisibleChk, x, y+rowH*4, leftW, 28, true)
		moveWindow(controls[6], x, y+rowH*5+6, labelW, 24, true)
		moveWindow(a.opacityEdit, x+labelW, y+rowH*5, 90, 32, true)
		moveWindow(controls[8], x, y+rowH*6+6, labelW, 24, true)
		moveWindow(a.bgPathEdit, x, y+rowH*7, leftW, 32, true)
		moveWindow(a.saveSettingsBtn, x, y+rowH*8+10, 120, 34, true)
	}
	moveWindow(a.settingsInfo, infoX, y, infoW, h, true)
}

func (a *uiApp) layoutHelp(x, y, w, h int32) {
	moveWindow(a.helpTitle, x, y, w, 34, true)
	moveWindow(a.helpText, x, y+46, w, h-104, true)
	moveWindow(a.openReadmeBtn, x, y+h-42, 120, 34, true)
}

func (a *uiApp) showPage(page string) {
	if page == "" {
		page = "study"
	}
	for name, controls := range a.pageCtrls {
		command := int32(swHide)
		if name == page {
			command = swShow
		}
		for _, control := range controls {
			showWindow(control, command)
		}
	}
	a.activePage = page
	a.layout()
	invalidateWindow(a.hwnd)
}

func (a *uiApp) handleControlColor(message uint32, hdc uintptr) uintptr {
	setTextColor(hdc, a.textColor)
	switch message {
	case wmCtlColorStatic:
		setBkMode(hdc, transparentBkMode)
		return getStockBrush(nullBrush)
	case wmCtlColorBtn:
		setBkMode(hdc, opaqueBkMode)
		setBkColor(hdc, a.buttonColor)
		return a.buttonBrush
	case wmCtlColorEdit, wmCtlColorListBox:
		setBkMode(hdc, opaqueBkMode)
		setBkColor(hdc, a.inputColor)
		return a.inputBrush
	default:
		return 0
	}
}

func (a *uiApp) handleCommand(id int, notify uint16) {
	switch id {
	case idOpenData:
		a.openPath(a.paths.DataDir)
	case idHide:
		a.hideWindow()
	case idQuit:
		a.quit()
	case idTrayShow:
		a.showWindowFromTray()
	case idTrayHide:
		a.hideWindow()
	case idTrayQuit:
		a.quit()
	case idNavStudy:
		a.showPage("study")
	case idNavWords:
		a.showPage("words")
	case idNavMemo:
		a.showPage("memo")
	case idNavPreview:
		a.showPage("preview")
	case idNavSettings:
		a.showPage("settings")
	case idNavHelp:
		a.showPage("help")
	case idReloadStudy, idReloadWords:
		a.reloadData()
	case idMarkUnknown:
		a.markCurrent("unknown")
	case idMarkKnown:
		a.markCurrent("known")
	case idPrevious:
		a.previousWord()
	case idNext:
		a.nextWord()
	case idSearch:
		if notify == enChange {
			a.refreshWordsList()
		}
	case idWordsList:
		if notify == lbnSelChange {
			a.refreshWordDetail()
		}
	case idAddWord:
		a.saveWordFromForm()
	case idChooseCSV:
		a.chooseCSV()
	case idImportCSV:
		a.importCSV()
	case idSaveMemo:
		a.saveMemo()
	case idOpenBackground:
		a.openBackground()
	case idChooseBackground:
		a.chooseBackground()
	case idResetBackground:
		a.resetBackground()
	case idSaveSettings:
		a.saveSettingsFromUI()
	case idOpenReadme:
		a.openPath(a.paths.ReadmePath)
	}
}

func (a *uiApp) handleTrayEvent(lParam uintptr) {
	switch uint32(lParam) {
	case wmLButtonUp, wmLButtonDblClk:
		a.showWindowFromTray()
	case wmRButtonUp:
		command := showTrayMenu(a.hwnd, idTrayShow, idTrayHide, idTrayQuit)
		if command != 0 {
			a.handleCommand(command, 0)
		}
	}
}

func (a *uiApp) refreshAll() {
	a.todayWords = BuildTodayWords(a.words, a.progress, a.settings.DailyGoal)
	a.loadBackgroundImage()
	a.refreshStudy()
	a.refreshWordsList()
	a.refreshPreview()
	a.refreshSettingsInfo()
	if a.helpText != 0 {
		setWindowText(a.helpText, a.helpContent())
	}
	invalidateWindow(a.hwnd)
}

func (a *uiApp) reloadData() {
	a.loadState()
	a.syncSettingsToControls()
	a.refreshAll()
	a.setStatus(fmt.Sprintf("已重新加载 %d 个单词。", len(a.words)))
}

func (a *uiApp) refreshStudy() {
	total, known, unknown, studiedToday := Summary(a.words, a.progress)
	setWindowText(a.metricTotal, fmt.Sprintf("总词数\r\n%d", total))
	setWindowText(a.metricKnown, fmt.Sprintf("已认识\r\n%d", known))
	setWindowText(a.metricUnknown, fmt.Sprintf("不认识\r\n%d", unknown))
	setWindowText(a.metricToday, fmt.Sprintf("今日已学\r\n%d", studiedToday))

	if len(a.todayWords) == 0 {
		setWindowText(a.studyCounter, fmt.Sprintf("今日已学习：%d / 目标 %d", studiedToday, a.settings.DailyGoal))
		setWindowText(a.wordLabel, "暂无单词")
		setWindowText(a.meaningLabel, "请在 data/words.csv 中添加单词，或在单词页导入 CSV。")
		setWindowText(a.exampleLabel, "")
		setWindowText(a.tagLabel, "")
		return
	}

	a.currentIndex = ClampInt(a.currentIndex, 0, len(a.todayWords)-1)
	word := a.todayWords[a.currentIndex]
	setWindowText(a.studyCounter, fmt.Sprintf("今日进度：%d / %d    今日已学习：%d / 目标 %d", a.currentIndex+1, len(a.todayWords), studiedToday, a.settings.DailyGoal))
	setWindowText(a.wordLabel, word.Word)
	setWindowText(a.meaningLabel, fallback(word.Meaning, "未填写释义"))
	setWindowText(a.exampleLabel, fallback(word.Example, "未填写例句"))
	setWindowText(a.tagLabel, "标签："+fallback(word.Tag, "无"))
}

func (a *uiApp) refreshWordsList() {
	if a.wordsList == 0 {
		return
	}
	resetListBox(a.wordsList)
	keyword := strings.ToLower(strings.TrimSpace(getWindowText(a.searchEdit)))
	a.filteredWords = a.filteredWords[:0]
	for _, word := range a.words {
		haystack := strings.ToLower(word.Word + " " + word.Meaning + " " + word.Tag)
		if keyword != "" && !strings.Contains(haystack, keyword) {
			continue
		}
		a.filteredWords = append(a.filteredWords, word)
		addListBoxString(a.wordsList, fmt.Sprintf("%-18s  %-18s  %-8s  %s", word.Word, word.Meaning, fallback(word.Tag, "-"), StatusLabel(WordStatus(a.progress, word.Key))))
	}
	if len(a.filteredWords) > 0 {
		selectListBox(a.wordsList, 0)
	}
	a.refreshWordDetail()
}

func (a *uiApp) refreshWordDetail() {
	index := listBoxSelected(a.wordsList)
	if index < 0 || index >= len(a.filteredWords) {
		setWindowText(a.wordDetail, "没有选中的单词。")
		return
	}
	word := a.filteredWords[index]
	item := a.progress.Words[word.Key]
	status := "未学习"
	lastStudied := ""
	knownCount := 0
	unknownCount := 0
	if item != nil {
		status = StatusLabel(item.Status)
		lastStudied = item.LastStudied
		knownCount = item.KnownCount
		unknownCount = item.UnknownCount
	}
	text := fmt.Sprintf(
		"单词：%s\r\n释义：%s\r\n标签：%s\r\n状态：%s\r\n上次学习：%s\r\n认识次数：%d\r\n不认识次数：%d\r\n\r\n例句：\r\n%s",
		word.Word,
		fallback(word.Meaning, "未填写"),
		fallback(word.Tag, "无"),
		status,
		fallback(lastStudied, "无"),
		knownCount,
		unknownCount,
		fallback(word.Example, "未填写"),
	)
	setWindowText(a.wordDetail, text)
}

func (a *uiApp) markCurrent(status string) {
	if len(a.todayWords) == 0 {
		return
	}
	word := a.todayWords[a.currentIndex]
	if err := MarkWord(a.paths, &a.progress, word, status); err != nil {
		a.errorStatus("保存学习进度失败", err)
		return
	}
	a.setStatus(fmt.Sprintf("已标记 %s：%s", StatusLabel(status), word.Word))
	if a.currentIndex < len(a.todayWords)-1 {
		a.currentIndex++
	}
	a.refreshStudy()
	a.refreshWordsList()
}

func (a *uiApp) previousWord() {
	if len(a.todayWords) == 0 {
		return
	}
	a.currentIndex = ClampInt(a.currentIndex-1, 0, len(a.todayWords)-1)
	a.refreshStudy()
}

func (a *uiApp) nextWord() {
	if len(a.todayWords) == 0 {
		return
	}
	a.currentIndex = ClampInt(a.currentIndex+1, 0, len(a.todayWords)-1)
	a.refreshStudy()
}

func (a *uiApp) saveWordFromForm() {
	word := getWindowText(a.addWordEdit)
	meaning := getWindowText(a.addMeaningEdit)
	tag := getWindowText(a.addTagEdit)
	example := getWindowText(a.addExampleEdit)

	result, err := AddOrUpdateWord(a.paths, word, meaning, example, tag, false)
	if err != nil {
		a.errorStatus("保存单词失败", err)
		return
	}
	if result == "duplicate" {
		result, err = AddOrUpdateWord(a.paths, word, meaning, example, tag, true)
		if err != nil {
			a.errorStatus("更新单词失败", err)
			return
		}
	}

	a.words = LoadWords(a.paths)
	a.todayWords = BuildTodayWords(a.words, a.progress, a.settings.DailyGoal)
	setWindowText(a.searchEdit, strings.TrimSpace(word))
	setWindowText(a.addWordEdit, "")
	setWindowText(a.addMeaningEdit, "")
	setWindowText(a.addTagEdit, "")
	setWindowText(a.addExampleEdit, "")
	a.refreshAll()
	if result == "updated" {
		a.setStatus("已更新单词：" + strings.TrimSpace(word))
	} else {
		a.setStatus("已添加单词：" + strings.TrimSpace(word))
	}
}

func (a *uiApp) chooseCSV() {
	path, ok := openFileDialog(a.hwnd, "选择单词 CSV 文件", "CSV 文件 (*.csv)\x00*.csv\x00所有文件 (*.*)\x00*.*\x00\x00")
	if ok {
		setWindowText(a.importPathEdit, path)
	}
}

func (a *uiApp) importCSV() {
	source := getWindowText(a.importPathEdit)
	if err := ImportWordsCSV(a.paths, source); err != nil {
		a.errorStatus("导入 CSV 失败", err)
		return
	}
	a.words = LoadWords(a.paths)
	a.todayWords = BuildTodayWords(a.words, a.progress, a.settings.DailyGoal)
	a.refreshAll()
	a.setStatus("已导入 CSV：" + source)
}

func (a *uiApp) saveMemo() {
	if err := WriteMemo(a.paths, getWindowText(a.memoEdit)); err != nil {
		a.errorStatus("保存备忘录失败", err)
		return
	}
	a.setStatus("备忘录已保存。")
}

func (a *uiApp) syncSettingsToControls() {
	setWindowText(a.dailyGoalEdit, strconv.Itoa(a.settings.DailyGoal))
	setCheckbox(a.startHiddenChk, a.settings.StartHidden)
	setCheckbox(a.autostartChk, isAutostartEnabled())
	setCheckbox(a.bgEnabledChk, a.settings.BackgroundEnabled)
	setCheckbox(a.bgVisibleChk, a.settings.BackgroundVisibleMode)
	setWindowText(a.opacityEdit, strconv.Itoa(a.settings.InterfaceOpacity))
	setWindowText(a.bgPathEdit, a.settings.BackgroundImage)
}

func (a *uiApp) saveSettingsFromUI() {
	dailyGoal, err := strconv.Atoi(strings.TrimSpace(getWindowText(a.dailyGoalEdit)))
	if err != nil {
		dailyGoal = a.settings.DailyGoal
	}
	opacity, err := strconv.Atoi(strings.TrimSpace(getWindowText(a.opacityEdit)))
	if err != nil {
		opacity = a.settings.InterfaceOpacity
	}

	autostart := checkboxChecked(a.autostartChk)
	if autostart != isAutostartEnabled() {
		if err := setAutostartEnabled(autostart); err != nil {
			a.errorStatus("设置开机自启动失败", err)
			setCheckbox(a.autostartChk, isAutostartEnabled())
			return
		}
	}

	a.settings.DailyGoal = ClampInt(dailyGoal, 1, 200)
	a.settings.StartHidden = checkboxChecked(a.startHiddenChk)
	a.settings.Autostart = autostart
	a.settings.BackgroundEnabled = checkboxChecked(a.bgEnabledChk)
	a.settings.BackgroundVisibleMode = checkboxChecked(a.bgVisibleChk)
	a.settings.InterfaceOpacity = ClampInt(opacity, 70, 100)
	a.settings.BackgroundImage = strings.TrimSpace(getWindowText(a.bgPathEdit))

	if err := SaveSettings(a.paths, a.settings); err != nil {
		a.errorStatus("保存设置失败", err)
		return
	}
	a.todayWords = BuildTodayWords(a.words, a.progress, a.settings.DailyGoal)
	a.loadBackgroundImage()
	a.syncSettingsToControls()
	a.refreshStudy()
	a.refreshPreview()
	a.refreshSettingsInfo()
	setWindowText(a.helpText, a.helpContent())
	a.layout()
	invalidateWindow(a.hwnd)
	a.setStatus("设置已保存。")
}

func (a *uiApp) refreshPreview() {
	path := ResolveAppPath(a.paths, a.settings.BackgroundImage)
	text := "当前背景图片：\r\n" + path + "\r\n\r\n"
	if !a.settings.BackgroundEnabled {
		text += "背景图当前处于关闭状态。\r\n"
	}
	if a.backgroundErr != "" {
		text += "背景图片加载失败：" + a.backgroundErr + "\r\n"
	} else if !exists(path) {
		text += "没有找到图片文件。\r\n"
	} else {
		text += fmt.Sprintf("图片已加载，尺寸：%d x %d。\r\n主窗口会按铺满方式显示背景。", a.backgroundWidth, a.backgroundHeight)
	}
	text += fmt.Sprintf("\r\n背景可见模式：%s\r\n界面淡化程度：%d%%", yesNo(a.settings.BackgroundVisibleMode), a.settings.InterfaceOpacity)
	setWindowText(a.previewText, text)
}

func (a *uiApp) openBackground() {
	path := ResolveAppPath(a.paths, getWindowText(a.bgPathEdit))
	if strings.TrimSpace(path) == "" {
		path = ResolveAppPath(a.paths, a.settings.BackgroundImage)
	}
	a.openPath(path)
}

func (a *uiApp) chooseBackground() {
	path, ok := openFileDialog(a.hwnd, "选择背景图片", "图片文件 (*.png;*.jpg;*.jpeg;*.webp;*.bmp)\x00*.png;*.jpg;*.jpeg;*.webp;*.bmp\x00所有文件 (*.*)\x00*.*\x00\x00")
	if !ok {
		return
	}
	rel, err := CopyBackgroundImage(a.paths, path)
	if err != nil {
		a.errorStatus("更换背景失败", err)
		return
	}
	setWindowText(a.bgPathEdit, rel)
	setCheckbox(a.bgEnabledChk, true)
	a.saveSettingsFromUI()
	a.setStatus("已更换背景：" + rel)
}

func (a *uiApp) resetBackground() {
	setWindowText(a.bgPathEdit, DefaultSettings().BackgroundImage)
	setCheckbox(a.bgEnabledChk, true)
	a.saveSettingsFromUI()
	a.setStatus("已恢复默认背景。")
}

func (a *uiApp) loadBackgroundImage() {
	a.backgroundPath = ResolveAppPath(a.paths, a.settings.BackgroundImage)
	a.backgroundPixels = nil
	a.backgroundWidth = 0
	a.backgroundHeight = 0
	a.backgroundErr = ""

	if !a.settings.BackgroundEnabled {
		return
	}

	file, err := os.Open(a.backgroundPath)
	if err != nil {
		a.backgroundErr = err.Error()
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		a.backgroundErr = "当前 Go 版支持 png、jpg、gif 背景预览；其他格式会保留路径但不绘制"
		return
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		a.backgroundErr = "图片尺寸无效"
		return
	}

	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)

	pixels := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			src := rgba.PixOffset(x, y)
			dst := (y*width + x) * 4
			pixels[dst+0] = rgba.Pix[src+2]
			pixels[dst+1] = rgba.Pix[src+1]
			pixels[dst+2] = rgba.Pix[src+0]
			pixels[dst+3] = 0
		}
	}

	a.backgroundPixels = pixels
	a.backgroundWidth = int32(width)
	a.backgroundHeight = int32(height)
}

func (a *uiApp) hasBackground() bool {
	return a.settings.BackgroundEnabled && len(a.backgroundPixels) > 0 && a.backgroundWidth > 0 && a.backgroundHeight > 0
}

func (a *uiApp) paintBackground() bool {
	if !a.hasBackground() {
		return false
	}

	hdc, ps := beginPaint(a.hwnd)
	defer endPaint(a.hwnd, &ps)

	rc := getClientRect(a.hwnd)
	destW := rc.Right - rc.Left
	destH := rc.Bottom - rc.Top
	if destW <= 0 || destH <= 0 {
		return true
	}

	srcW := a.backgroundWidth
	srcH := a.backgroundHeight
	cropW := srcW
	cropH := int32(float64(destH) * float64(srcW) / float64(destW))
	if cropH > srcH {
		cropH = srcH
		cropW = int32(float64(destW) * float64(srcH) / float64(destH))
	}
	if cropW <= 0 || cropH <= 0 {
		return true
	}
	srcX := (srcW - cropW) / 2
	srcY := (srcH - cropH) / 2

	stretchDIBits(hdc, destW, destH, srcX, srcY, cropW, cropH, a.backgroundPixels, srcW, srcH)
	return true
}

func (a *uiApp) refreshSettingsInfo() {
	info := fmt.Sprintf(
		"数据目录：%s\r\n单词文件：%s\r\n进度文件：%s\r\n备忘录：%s\r\n\r\n背景启用：%s\r\n背景文件：%s\r\n背景可见模式：%s\r\n界面淡化程度：%d%%\r\n\r\n开机自启动：%s\r\n自启动命令：%s",
		a.paths.DataDir,
		a.paths.WordsPath,
		a.paths.ProgressPath,
		a.paths.MemoPath,
		yesNo(a.settings.BackgroundEnabled),
		ResolveAppPath(a.paths, a.settings.BackgroundImage),
		yesNo(a.settings.BackgroundVisibleMode),
		a.settings.InterfaceOpacity,
		yesNo(isAutostartEnabled()),
		BuildAutostartCommand(),
	)
	setWindowText(a.settingsInfo, info)
}

func (a *uiApp) helpContent() string {
	return fmt.Sprintf(
		"Desktop Helper Go 版 "+appVersion+"\r\n\r\n"+
			"这个界面是 Windows 原生 Go 版本，继续使用项目目录内的数据文件，不需要登录、云同步或数据库服务。\r\n\r\n"+
			"常用操作\r\n"+
			"1. 今日学习：点击“认识”或“不认识”会写入 data/progress.json。\r\n"+
			"2. 单词列表：可以搜索单词、手动添加单词，也可以导入 CSV。\r\n"+
			"3. 备忘录：输入纯文本后点击“保存备忘录”，内容会写入 data/memo.txt。\r\n"+
			"4. 背景预览：可以打开、更换或恢复背景图；新背景会复制到 assets/。\r\n"+
			"5. 设置：可以调整每日学习数量、启动后隐藏、开机自启动、背景可见模式和界面淡化程度。\r\n"+
			"6. 关闭窗口：优先隐藏到系统托盘；托盘右键菜单可以显示、隐藏或退出。\r\n\r\n"+
			"CSV 格式\r\n"+
			"word,meaning,example,tag\r\n"+
			"apple,苹果,An apple a day keeps the doctor away.,basic\r\n\r\n"+
			"数据位置\r\n"+
			"单词：%s\r\n"+
			"进度：%s\r\n"+
			"备忘录：%s\r\n"+
			"设置：%s\r\n"+
			"背景：%s\r\n\r\n"+
			"运行命令\r\n"+
			"go run ./cmd/desktop-helper\r\n\r\n"+
			"构建命令\r\n"+
			"go build -ldflags=\"-H=windowsgui\" -o build/desktop-helper-go.exe ./cmd/desktop-helper\r\n",
		a.paths.WordsPath,
		a.paths.ProgressPath,
		a.paths.MemoPath,
		a.paths.SettingsPath,
		ResolveAppPath(a.paths, a.settings.BackgroundImage),
	)
}

func (a *uiApp) openPath(path string) {
	if err := shellOpen(a.hwnd, path); err != nil {
		a.errorStatus("打开失败", err)
	}
}

func (a *uiApp) hideWindow() {
	a.saveMemo()
	a.saveSettingsFromUI()
	if a.trayIconAdded {
		a.setStatus("窗口已隐藏到系统托盘。")
		showWindow(a.hwnd, swHide)
		return
	}
	showWindow(a.hwnd, swMinimize)
	a.setStatus("托盘图标不可用，窗口已最小化到任务栏。")
}

func (a *uiApp) showWindowFromTray() {
	showWindow(a.hwnd, swRestore)
	showWindow(a.hwnd, swShow)
	setForegroundWindow(a.hwnd)
	a.setStatus("窗口已显示。")
}

func (a *uiApp) quit() {
	if a.isQuitting {
		return
	}
	a.isQuitting = true
	_ = WriteMemo(a.paths, getWindowText(a.memoEdit))
	_ = SaveSettings(a.paths, a.settings)
	if a.trayIconAdded {
		deleteTrayIcon(a.hwnd)
		a.trayIconAdded = false
	}
	showWindow(a.hwnd, swHide)
	destroyWindow(a.hwnd)
	postQuitMessage(0)
}

func (a *uiApp) cleanup() {
	if a.trayIconAdded {
		deleteTrayIcon(a.hwnd)
		a.trayIconAdded = false
	}
	deleteObject(a.titleFont)
	deleteObject(a.wordFont)
	deleteObject(a.monoFont)
	deleteObject(a.inputBrush)
	deleteObject(a.buttonBrush)
}

func (a *uiApp) setStatus(text string) {
	setWindowText(a.statusLabel, text)
}

func (a *uiApp) errorStatus(title string, err error) {
	text := title + "：" + err.Error()
	a.setStatus(text)
	messageBox(a.hwnd, title, err.Error())
}

func fallback(value, fallbackValue string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallbackValue
	}
	return value
}

func yesNo(value bool) string {
	if value {
		return "开启"
	}
	return "关闭"
}
