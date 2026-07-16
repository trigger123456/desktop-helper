package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testPaths(t *testing.T) Paths {
	t.Helper()
	root := t.TempDir()
	return Paths{
		AppDir:            root,
		AssetsDir:         filepath.Join(root, "assets"),
		DataDir:           filepath.Join(root, "data"),
		WordsPath:         filepath.Join(root, "data", "words.csv"),
		ProgressPath:      filepath.Join(root, "data", "progress.json"),
		MemoPath:          filepath.Join(root, "data", "memo.txt"),
		SettingsPath:      filepath.Join(root, "data", "settings.json"),
		ReadmePath:        filepath.Join(root, "README.md"),
		DefaultBackground: filepath.Join(root, "assets", "background.png"),
	}
}

func TestEnsureDataFilesCreatesProjectData(t *testing.T) {
	paths := testPaths(t)

	if err := EnsureDataFiles(paths); err != nil {
		t.Fatalf("EnsureDataFiles failed: %v", err)
	}

	for _, path := range []string{paths.WordsPath, paths.ProgressPath, paths.MemoPath, paths.SettingsPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	words := LoadWords(paths)
	if len(words) != 3 {
		t.Fatalf("expected sample words, got %d", len(words))
	}
	if words[0].Word != "apple" || words[0].Meaning != "苹果" {
		t.Fatalf("sample words not loaded as UTF-8: %#v", words[0])
	}
}

func TestAddOrUpdateWordAndLoadWords(t *testing.T) {
	paths := testPaths(t)
	if err := EnsureDataFiles(paths); err != nil {
		t.Fatalf("EnsureDataFiles failed: %v", err)
	}

	result, err := AddOrUpdateWord(paths, "  Plan  ", "计划", "Make a plan.", "daily", false)
	if err != nil || result != "added" {
		t.Fatalf("expected added, got result=%q err=%v", result, err)
	}

	result, err = AddOrUpdateWord(paths, "plan", "方案", "Update the plan.", "work", false)
	if err != nil || result != "duplicate" {
		t.Fatalf("expected duplicate, got result=%q err=%v", result, err)
	}

	result, err = AddOrUpdateWord(paths, "plan", "方案", "Update the plan.", "work", true)
	if err != nil || result != "updated" {
		t.Fatalf("expected updated, got result=%q err=%v", result, err)
	}

	words := LoadWords(paths)
	found := false
	for _, word := range words {
		if word.Key == "plan" {
			found = true
			if word.Word != "plan" || word.Meaning != "方案" || word.Tag != "work" {
				t.Fatalf("updated word mismatch: %#v", word)
			}
		}
	}
	if !found {
		t.Fatal("updated word was not loaded")
	}
}

func TestImportWordsCSVRewritesWordsFile(t *testing.T) {
	paths := testPaths(t)
	if err := EnsureDataFiles(paths); err != nil {
		t.Fatalf("EnsureDataFiles failed: %v", err)
	}

	source := filepath.Join(t.TempDir(), "words.csv")
	content := "\ufeffword,meaning,example,tag\nriver,河流,The river is wide.,nature\nsky,天空,,basic\n"
	if err := os.WriteFile(source, []byte(content), 0644); err != nil {
		t.Fatalf("write import file: %v", err)
	}

	if err := ImportWordsCSV(paths, source); err != nil {
		t.Fatalf("ImportWordsCSV failed: %v", err)
	}

	words := LoadWords(paths)
	if len(words) != 2 {
		t.Fatalf("expected 2 imported words, got %d", len(words))
	}
	if words[0].Word != "river" || words[1].Word != "sky" {
		t.Fatalf("unexpected imported words: %#v", words)
	}
}

func TestProgressMemoAndSettingsRoundTrip(t *testing.T) {
	paths := testPaths(t)
	if err := EnsureDataFiles(paths); err != nil {
		t.Fatalf("EnsureDataFiles failed: %v", err)
	}

	words := LoadWords(paths)
	progress := LoadProgress(paths)
	if err := MarkWord(paths, &progress, words[0], "known"); err != nil {
		t.Fatalf("MarkWord failed: %v", err)
	}

	reloadedProgress := LoadProgress(paths)
	item := reloadedProgress.Words[words[0].Key]
	if item == nil || item.Status != "known" || item.KnownCount != 1 || item.LastStudied == "" {
		t.Fatalf("progress not persisted: %#v", item)
	}

	if err := WriteMemo(paths, "今天复习 10 个词"); err != nil {
		t.Fatalf("WriteMemo failed: %v", err)
	}
	if got := ReadMemo(paths); got != "今天复习 10 个词" {
		t.Fatalf("memo mismatch: %q", got)
	}

	settings := LoadSettings(paths)
	settings.DailyGoal = 999
	settings.InterfaceOpacity = 12
	settings.BackgroundImage = ""
	if err := SaveSettings(paths, settings); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}
	reloadedSettings := LoadSettings(paths)
	if reloadedSettings.DailyGoal != 200 {
		t.Fatalf("daily goal was not clamped: %d", reloadedSettings.DailyGoal)
	}
	if reloadedSettings.InterfaceOpacity != 70 {
		t.Fatalf("interface opacity was not clamped: %d", reloadedSettings.InterfaceOpacity)
	}
	if reloadedSettings.BackgroundImage != DefaultSettings().BackgroundImage {
		t.Fatalf("background fallback mismatch: %q", reloadedSettings.BackgroundImage)
	}
}

func TestCopyBackgroundImageStoresRelativeAssetPath(t *testing.T) {
	paths := testPaths(t)
	if err := EnsureDataFiles(paths); err != nil {
		t.Fatalf("EnsureDataFiles failed: %v", err)
	}

	source := filepath.Join(t.TempDir(), "cover.jpg")
	if err := os.WriteFile(source, []byte("fake image bytes"), 0644); err != nil {
		t.Fatalf("write background source: %v", err)
	}

	rel, err := CopyBackgroundImage(paths, source)
	if err != nil {
		t.Fatalf("CopyBackgroundImage failed: %v", err)
	}
	if rel != "assets/background_custom.jpg" {
		t.Fatalf("expected relative asset path, got %q", rel)
	}
	target := ResolveAppPath(paths, rel)
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read copied background: %v", err)
	}
	if string(data) != "fake image bytes" {
		t.Fatalf("copied background data mismatch: %q", string(data))
	}
}

func TestWriteJSONKeepsUTF8Content(t *testing.T) {
	paths := testPaths(t)
	progress := Progress{
		Words: map[string]*ProgressItem{
			"memory": {
				Word:        "memory",
				Status:      "unknown",
				LastStudied: "2026-07-16",
				UpdatedAt:   "2026-07-16 10:20:30",
			},
		},
	}
	if err := SaveProgress(paths, progress); err != nil {
		t.Fatalf("SaveProgress failed: %v", err)
	}

	raw, err := os.ReadFile(paths.ProgressPath)
	if err != nil {
		t.Fatalf("read progress: %v", err)
	}
	if strings.Contains(string(raw), `\u`) {
		t.Fatalf("progress json should keep readable UTF-8, got %s", raw)
	}

	var decoded Progress
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("progress json invalid: %v", err)
	}
}
