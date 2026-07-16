package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	appName    = "DesktopHelper"
	appVersion = "v2.0"
)

type Word struct {
	Key     string
	Word    string
	Meaning string
	Example string
	Tag     string
}

type WordRow struct {
	Word    string
	Meaning string
	Example string
	Tag     string
}

type Progress struct {
	Words map[string]*ProgressItem `json:"words"`
}

type ProgressItem struct {
	Word         string `json:"word"`
	Status       string `json:"status"`
	LastStudied  string `json:"last_studied"`
	UpdatedAt    string `json:"updated_at"`
	KnownCount   int    `json:"known_count,omitempty"`
	UnknownCount int    `json:"unknown_count,omitempty"`
}

type Settings struct {
	DailyGoal             int    `json:"daily_goal"`
	StartHidden           bool   `json:"start_hidden"`
	Autostart             bool   `json:"autostart"`
	WindowGeometry        string `json:"window_geometry"`
	BackgroundEnabled     bool   `json:"background_enabled"`
	BackgroundImage       string `json:"background_image"`
	BackgroundVisibleMode bool   `json:"background_visible_mode"`
	InterfaceOpacity      int    `json:"interface_opacity"`
}

type Paths struct {
	AppDir            string
	AssetsDir         string
	DataDir           string
	WordsPath         string
	ProgressPath      string
	MemoPath          string
	SettingsPath      string
	ReadmePath        string
	DefaultBackground string
}

func DefaultSettings() Settings {
	return Settings{
		DailyGoal:             10,
		StartHidden:           false,
		Autostart:             false,
		WindowGeometry:        "1040x680",
		BackgroundEnabled:     true,
		BackgroundImage:       "assets/background.png",
		BackgroundVisibleMode: false,
		InterfaceOpacity:      100,
	}
}

func SampleWords() []WordRow {
	return []WordRow{
		{Word: "apple", Meaning: "苹果", Example: "An apple a day keeps the doctor away.", Tag: "basic"},
		{Word: "abandon", Meaning: "放弃", Example: "Do not abandon your plan.", Tag: "cet4"},
		{Word: "focus", Meaning: "专注", Example: "Focus on one thing at a time.", Tag: "study"},
	}
}

func TodayText() string {
	return time.Now().Format("2006-01-02")
}

func NowText() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func ClampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func AppPaths() Paths {
	appDir := detectAppDir()
	return Paths{
		AppDir:            appDir,
		AssetsDir:         filepath.Join(appDir, "assets"),
		DataDir:           filepath.Join(appDir, "data"),
		WordsPath:         filepath.Join(appDir, "data", "words.csv"),
		ProgressPath:      filepath.Join(appDir, "data", "progress.json"),
		MemoPath:          filepath.Join(appDir, "data", "memo.txt"),
		SettingsPath:      filepath.Join(appDir, "data", "settings.json"),
		ReadmePath:        filepath.Join(appDir, "README.md"),
		DefaultBackground: filepath.Join(appDir, "assets", "background.png"),
	}
}

func detectAppDir() string {
	cwd, err := os.Getwd()
	if err == nil {
		if exists(filepath.Join(cwd, "data")) || exists(filepath.Join(cwd, "go.mod")) || exists(filepath.Join(cwd, "main.py")) {
			return cwd
		}
	}

	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		if exists(filepath.Join(dir, "data")) || exists(filepath.Join(dir, "assets")) {
			return dir
		}
	}

	if err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func EnsureDataFiles(paths Paths) error {
	if err := os.MkdirAll(paths.AssetsDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(paths.DataDir, 0755); err != nil {
		return err
	}

	if !exists(paths.WordsPath) {
		if err := WriteWordRows(paths, SampleWords()); err != nil {
			return err
		}
	}
	if !exists(paths.ProgressPath) {
		if err := SaveProgress(paths, Progress{Words: map[string]*ProgressItem{}}); err != nil {
			return err
		}
	}
	if !exists(paths.MemoPath) {
		if err := os.WriteFile(paths.MemoPath, []byte(""), 0644); err != nil {
			return err
		}
	}
	if !exists(paths.SettingsPath) {
		if err := SaveSettings(paths, DefaultSettings()); err != nil {
			return err
		}
	}

	return nil
}

func LoadSettings(paths Paths) Settings {
	settings := DefaultSettings()
	file, err := os.Open(paths.SettingsPath)
	if err != nil {
		return settings
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&settings); err != nil {
		return DefaultSettings()
	}

	settings.DailyGoal = ClampInt(settings.DailyGoal, 1, 200)
	if strings.TrimSpace(settings.WindowGeometry) == "" {
		settings.WindowGeometry = DefaultSettings().WindowGeometry
	}
	if strings.TrimSpace(settings.BackgroundImage) == "" {
		settings.BackgroundImage = DefaultSettings().BackgroundImage
	}
	if settings.InterfaceOpacity == 0 {
		settings.InterfaceOpacity = 100
	}
	settings.InterfaceOpacity = ClampInt(settings.InterfaceOpacity, 70, 100)
	return settings
}

func SaveSettings(paths Paths, settings Settings) error {
	settings.DailyGoal = ClampInt(settings.DailyGoal, 1, 200)
	settings.InterfaceOpacity = ClampInt(settings.InterfaceOpacity, 70, 100)
	if settings.BackgroundImage == "" {
		settings.BackgroundImage = DefaultSettings().BackgroundImage
	}
	return writeJSON(paths.SettingsPath, settings)
}

func LoadProgress(paths Paths) Progress {
	progress := Progress{Words: map[string]*ProgressItem{}}
	file, err := os.Open(paths.ProgressPath)
	if err != nil {
		return progress
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&progress); err != nil || progress.Words == nil {
		progress.Words = map[string]*ProgressItem{}
	}
	return progress
}

func SaveProgress(paths Paths, progress Progress) error {
	if progress.Words == nil {
		progress.Words = map[string]*ProgressItem{}
	}
	return writeJSON(paths.ProgressPath, progress)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func WordKey(word string) string {
	return strings.ToLower(strings.TrimSpace(word))
}

func LoadWords(paths Paths) []Word {
	rows, err := ReadWordRows(paths)
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	words := make([]Word, 0, len(rows))
	for _, row := range rows {
		cleanWord := strings.TrimSpace(row.Word)
		if cleanWord == "" {
			continue
		}
		key := WordKey(cleanWord)
		if seen[key] {
			continue
		}
		seen[key] = true
		words = append(words, Word{
			Key:     key,
			Word:    cleanWord,
			Meaning: strings.TrimSpace(row.Meaning),
			Example: strings.TrimSpace(row.Example),
			Tag:     strings.TrimSpace(row.Tag),
		})
	}
	return words
}

func ReadWordRows(paths Paths) ([]WordRow, error) {
	file, err := os.Open(paths.WordsPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}

	columns := map[string]int{}
	for index, name := range header {
		columns[strings.ToLower(strings.TrimSpace(name))] = index
	}

	var rows []WordRow
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return rows, err
		}
		rows = append(rows, WordRow{
			Word:    csvValue(record, columns, "word"),
			Meaning: csvValue(record, columns, "meaning"),
			Example: csvValue(record, columns, "example"),
			Tag:     csvValue(record, columns, "tag"),
		})
	}
	return rows, nil
}

func csvValue(record []string, columns map[string]int, name string) string {
	index, ok := columns[name]
	if !ok || index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func WriteWordRows(paths Paths, rows []WordRow) error {
	if err := os.MkdirAll(filepath.Dir(paths.WordsPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(paths.WordsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}

	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"word", "meaning", "example", "tag"}); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writer.Write([]string{
			strings.TrimSpace(row.Word),
			strings.TrimSpace(row.Meaning),
			strings.TrimSpace(row.Example),
			strings.TrimSpace(row.Tag),
		}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func AddOrUpdateWord(paths Paths, word, meaning, example, tag string, updateExisting bool) (string, error) {
	cleanWord := strings.TrimSpace(word)
	if cleanWord == "" {
		return "", errors.New("请先填写单词")
	}

	rows, err := ReadWordRows(paths)
	if err != nil {
		return "", err
	}

	key := WordKey(cleanWord)
	newRow := WordRow{
		Word:    cleanWord,
		Meaning: strings.TrimSpace(meaning),
		Example: strings.TrimSpace(example),
		Tag:     strings.TrimSpace(tag),
	}

	for index, row := range rows {
		if WordKey(row.Word) == key {
			if !updateExisting {
				return "duplicate", nil
			}
			rows[index] = newRow
			if err := WriteWordRows(paths, rows); err != nil {
				return "", err
			}
			return "updated", nil
		}
	}

	rows = append(rows, newRow)
	if err := WriteWordRows(paths, rows); err != nil {
		return "", err
	}
	return "added", nil
}

func ImportWordsCSV(paths Paths, source string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return errors.New("请先选择 CSV 文件")
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if !utf8.Valid(data) {
		return errors.New("CSV 文件需要使用 UTF-8 编码")
	}

	tmp := filepath.Join(paths.DataDir, ".words-import-check.csv")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	defer os.Remove(tmp)

	checkPaths := paths
	checkPaths.WordsPath = tmp
	rows, err := ReadWordRows(checkPaths)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return errors.New("CSV 文件没有可导入的单词")
	}
	return WriteWordRows(paths, rows)
}

func ReadMemo(paths Paths) string {
	data, err := os.ReadFile(paths.MemoPath)
	if err != nil {
		return ""
	}
	return string(data)
}

func WriteMemo(paths Paths, text string) error {
	if err := os.MkdirAll(filepath.Dir(paths.MemoPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(paths.MemoPath, []byte(text), 0644)
}

func StatusLabel(status string) string {
	switch status {
	case "known":
		return "认识"
	case "unknown":
		return "不认识"
	default:
		return "未学习"
	}
}

func BuildTodayWords(words []Word, progress Progress, dailyGoal int) []Word {
	goal := ClampInt(dailyGoal, 1, 200)
	source := make([]Word, 0, len(words))
	for _, word := range words {
		if WordStatus(progress, word.Key) != "known" {
			source = append(source, word)
		}
	}
	if len(source) == 0 {
		source = words
	}
	if len(source) > goal {
		source = source[:goal]
	}
	return append([]Word(nil), source...)
}

func WordStatus(progress Progress, key string) string {
	if progress.Words == nil {
		return "new"
	}
	item := progress.Words[key]
	if item == nil || item.Status == "" {
		return "new"
	}
	return item.Status
}

func MarkWord(paths Paths, progress *Progress, word Word, status string) error {
	if progress.Words == nil {
		progress.Words = map[string]*ProgressItem{}
	}
	item := progress.Words[word.Key]
	if item == nil {
		item = &ProgressItem{}
		progress.Words[word.Key] = item
	}
	item.Word = word.Word
	item.Status = status
	item.LastStudied = TodayText()
	item.UpdatedAt = NowText()
	if status == "known" {
		item.KnownCount++
	} else {
		item.UnknownCount++
	}
	return SaveProgress(paths, *progress)
}

func Summary(words []Word, progress Progress) (total, known, unknown, studiedToday int) {
	total = len(words)
	for _, word := range words {
		item := progress.Words[word.Key]
		if item == nil {
			continue
		}
		switch item.Status {
		case "known":
			known++
		case "unknown":
			unknown++
		}
		if item.LastStudied == TodayText() {
			studiedToday++
		}
	}
	return total, known, unknown, studiedToday
}

func ResolveAppPath(paths Paths, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = DefaultSettings().BackgroundImage
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(paths.AppDir, filepath.FromSlash(value))
}

func AppRelativePath(paths Paths, path string) string {
	rel, err := filepath.Rel(paths.AppDir, path)
	if err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
		return filepath.ToSlash(rel)
	}
	return path
}

func CopyBackgroundImage(paths Paths, source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", errors.New("请先选择背景图片")
	}
	info, err := os.Stat(source)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("请选择图片文件，不要选择文件夹")
	}

	ext := strings.ToLower(filepath.Ext(source))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".bmp":
	default:
		ext = ".png"
	}

	if err := os.MkdirAll(paths.AssetsDir, 0755); err != nil {
		return "", err
	}
	target := filepath.Join(paths.AssetsDir, "background_custom"+ext)
	if sameFile(source, target) {
		return AppRelativePath(paths, target), nil
	}
	if err := copyFile(source, target); err != nil {
		return "", err
	}
	return AppRelativePath(paths, target), nil
}

func sameFile(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	return strings.EqualFold(absA, absB)
}

func copyFile(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(target)
	if err != nil {
		return err
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	return output.Close()
}

func BuildAutostartCommand() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%q --hidden", exe)
}
