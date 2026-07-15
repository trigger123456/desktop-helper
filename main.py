from __future__ import annotations

import csv
import json
import os
import platform
import shutil
import sys
import threading
from datetime import datetime
from pathlib import Path
from tkinter import (
    BOTH,
    END,
    LEFT,
    RIGHT,
    TOP,
    BOTTOM,
    X,
    Y,
    BooleanVar,
    DoubleVar,
    IntVar,
    Label,
    StringVar,
    Tk,
    Toplevel,
    Text,
    messagebox,
)
from tkinter import filedialog, ttk

try:
    import winreg
except ImportError:  # pragma: no cover - Windows-only module.
    winreg = None

try:
    import pystray
except ImportError:  # Optional dependency. The app still works without tray.
    pystray = None

try:
    from PIL import Image, ImageDraw, ImageOps, ImageTk
except ImportError:  # Optional dependency. Background and tray icons degrade without it.
    Image = None
    ImageDraw = None
    ImageOps = None
    ImageTk = None


APP_NAME = "DesktopHelper"
RUN_REGISTRY_PATH = r"Software\Microsoft\Windows\CurrentVersion\Run"

THEME = {
    "window": "#eef2f6",
    "surface": "#f8fafc",
    "card": "#ffffff",
    "surface_visible": "#fff7fb",
    "card_visible": "#fffafc",
    "text": "#172033",
    "muted": "#5d6678",
    "line": "#d8dee8",
    "primary": "#2563eb",
    "primary_hover": "#1d4ed8",
    "danger": "#dc2626",
    "danger_hover": "#b91c1c",
    "success": "#15803d",
    "warning": "#b45309",
    "input": "#ffffff",
}

APP_FONT = ("Microsoft YaHei UI", 10)
TITLE_FONT = ("Microsoft YaHei UI", 18, "bold")
SUBTITLE_FONT = ("Microsoft YaHei UI", 10)
WORD_FONT = ("Segoe UI", 34, "bold")
MEANING_FONT = ("Microsoft YaHei UI", 16)
MONO_FONT = ("Consolas", 10)


def blend_hex_color(foreground: str, background: str, percent: int) -> str:
    percent = clamp_int(percent, 0, 100)
    ratio = percent / 100
    fg = foreground.lstrip("#")
    bg = background.lstrip("#")
    channels = []
    for index in (0, 2, 4):
        fg_value = int(fg[index : index + 2], 16)
        bg_value = int(bg[index : index + 2], 16)
        mixed = round(fg_value * ratio + bg_value * (1 - ratio))
        channels.append(f"{mixed:02x}")
    return "#" + "".join(channels)


def get_app_dir() -> Path:
    if getattr(sys, "frozen", False):
        return Path(sys.executable).resolve().parent
    return Path(__file__).resolve().parent


def configure_app_style(root: Tk, visible_mode: bool = False, interface_opacity: int = 100) -> None:
    window = THEME["window"]
    interface_opacity = clamp_int(interface_opacity, 70, 100)
    surface_base = THEME["surface_visible"] if visible_mode else THEME["surface"]
    card_base = THEME["card_visible"] if visible_mode else THEME["card"]
    surface = blend_hex_color(surface_base, window, interface_opacity)
    card = blend_hex_color(card_base, window, interface_opacity)

    root.configure(bg=window)

    style = ttk.Style(root)
    try:
        style.theme_use("clam")
    except Exception:
        pass

    style.configure(".", font=APP_FONT, background=window, foreground=THEME["text"])
    style.configure("App.TFrame", background=window)
    style.configure("Surface.TFrame", background=surface)
    style.configure("Card.TFrame", background=card, relief="solid", borderwidth=1)
    style.configure("Header.TFrame", background=surface)

    style.configure("TLabel", background=window, foreground=THEME["text"])
    style.configure("Surface.TLabel", background=surface, foreground=THEME["text"])
    style.configure("Card.TLabel", background=card, foreground=THEME["text"])
    style.configure("Title.TLabel", background=surface, foreground=THEME["text"], font=TITLE_FONT)
    style.configure("Subtitle.TLabel", background=surface, foreground=THEME["muted"], font=SUBTITLE_FONT)
    style.configure("Muted.TLabel", background=window, foreground=THEME["muted"])
    style.configure("CardMuted.TLabel", background=card, foreground=THEME["muted"])
    style.configure("MetricValue.TLabel", background=card, foreground=THEME["text"], font=("Segoe UI", 18, "bold"))
    style.configure("MetricTitle.TLabel", background=card, foreground=THEME["muted"], font=("Microsoft YaHei UI", 9))
    style.configure("Word.TLabel", background=card, foreground=THEME["text"], font=WORD_FONT)
    style.configure("Meaning.TLabel", background=card, foreground=THEME["text"], font=MEANING_FONT)
    style.configure("Status.TLabel", background=surface, foreground=THEME["muted"], padding=(8, 6))

    style.configure("TButton", padding=(14, 7), borderwidth=0, background="#e4e9f2", foreground=THEME["text"])
    style.map("TButton", background=[("active", "#d8deea")])
    style.configure("Accent.TButton", background=THEME["primary"], foreground="#ffffff")
    style.map("Accent.TButton", background=[("active", THEME["primary_hover"])], foreground=[("active", "#ffffff")])
    style.configure("Danger.TButton", background=THEME["danger"], foreground="#ffffff")
    style.map("Danger.TButton", background=[("active", THEME["danger_hover"])], foreground=[("active", "#ffffff")])
    style.configure("Success.TButton", background=THEME["success"], foreground="#ffffff")
    style.map("Success.TButton", background=[("active", "#166534")], foreground=[("active", "#ffffff")])

    style.configure("TEntry", fieldbackground=THEME["input"], bordercolor=THEME["line"], lightcolor=THEME["line"], darkcolor=THEME["line"], padding=6)
    style.configure("TSpinbox", fieldbackground=THEME["input"], bordercolor=THEME["line"], lightcolor=THEME["line"], darkcolor=THEME["line"], padding=6)
    style.configure("TCheckbutton", background=window, foreground=THEME["text"])
    style.configure("Surface.TCheckbutton", background=surface, foreground=THEME["text"])

    style.configure("TNotebook", background=window, borderwidth=0)
    style.configure("TNotebook.Tab", padding=(18, 9), background="#dfe5ee", foreground=THEME["muted"])
    style.map(
        "TNotebook.Tab",
        background=[("selected", THEME["card"]), ("active", "#edf1f7")],
        foreground=[("selected", THEME["text"]), ("active", THEME["text"])],
    )

    style.configure(
        "Treeview",
        rowheight=32,
        background=card,
        fieldbackground=card,
        foreground=THEME["text"],
        bordercolor=THEME["line"],
        lightcolor=THEME["line"],
        darkcolor=THEME["line"],
    )
    style.configure("Treeview.Heading", background="#edf1f7", foreground=THEME["muted"], font=("Microsoft YaHei UI", 9, "bold"))
    style.map("Treeview", background=[("selected", "#dbeafe")], foreground=[("selected", THEME["text"])])


APP_DIR = get_app_dir()
ASSETS_DIR = APP_DIR / "assets"
DEFAULT_BACKGROUND_PATH = ASSETS_DIR / "background.png"
README_PATH = APP_DIR / "README.md"
DATA_DIR = APP_DIR / "data"
WORDS_PATH = DATA_DIR / "words.csv"
PROGRESS_PATH = DATA_DIR / "progress.json"
MEMO_PATH = DATA_DIR / "memo.txt"
SETTINGS_PATH = DATA_DIR / "settings.json"

DEFAULT_SETTINGS = {
    "daily_goal": 10,
    "start_hidden": False,
    "autostart": False,
    "window_geometry": "1040x680",
    "background_enabled": True,
    "background_image": "assets/background.png",
    "background_visible_mode": False,
    "interface_opacity": 100,
}

SAMPLE_WORDS = [
    {
        "word": "apple",
        "meaning": "苹果",
        "example": "An apple a day keeps the doctor away.",
        "tag": "basic",
    },
    {
        "word": "abandon",
        "meaning": "放弃",
        "example": "Do not abandon your plan.",
        "tag": "cet4",
    },
    {
        "word": "focus",
        "meaning": "专注",
        "example": "Focus on one thing at a time.",
        "tag": "study",
    },
]

USAGE_TEXT = """使用说明

1. 启动程序
   双击打包后的 exe，或在项目目录执行 python main.py。首次启动会自动创建 data 文件夹和示例数据。

2. 导入单词
   打开“单词列表”页面，点击“添加单词”可以手动新增一个单词。
   如果有现成单词文件，也可以点击“导入 CSV”，选择你的单词文件。CSV 表头必须包含 word,meaning,example,tag。
   其中 word 是必填，meaning、example、tag 可以留空。重复单词会提示是否更新。

3. 学习单词
   打开“今日学习”页面，程序会按“每日学习数量”挑选未认识的单词。点击“认识”或“不认识”后，进度会保存到 data/progress.json。

4. 查看和搜索单词
   打开“单词列表”页面，可以查看全部单词、学习状态和上次学习日期。搜索框支持按单词、释义或标签筛选。

5. 使用备忘录
   打开“备忘录”页面，直接输入文字，点击“保存备忘录”。内容保存到 data/memo.txt。

6. 修改设置
   打开“设置”页面，可以修改每日学习数量、开启或关闭开机自启动、设置启动后自动隐藏。
   也可以启用/关闭背景图，或点击“更换背景图”选择新的图片。图片会复制到 assets 文件夹，方便迁移。
   如果背景被界面遮住，可以打开“背景可见模式”，也可以调整“界面淡化程度”。背景图不会跟着淡化。

7. 托盘与隐藏
   如果安装了 pystray 和 Pillow，点击“隐藏窗口”或关闭窗口会隐藏到系统托盘；从托盘菜单可以显示或退出。
   如果没有安装托盘依赖，关闭窗口会最小化到任务栏。

8. 管理数据
   点击顶部“打开数据文件夹”可以查看 words.csv、progress.json、memo.txt、settings.json。备份整个 data 文件夹即可保留数据。

9. 背景预览
   打开“背景预览”页面可以单独欣赏当前背景图，不会被学习界面的白色面板遮挡。
   这个页面也可以更换背景图、恢复默认背景或打开图片文件。
"""


def today_text() -> str:
    return datetime.now().strftime("%Y-%m-%d")


def now_text() -> str:
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")


def clamp_int(value: int, minimum: int, maximum: int) -> int:
    return max(minimum, min(value, maximum))


def ensure_data_files() -> None:
    ASSETS_DIR.mkdir(exist_ok=True)
    DATA_DIR.mkdir(exist_ok=True)

    if not WORDS_PATH.exists():
        with WORDS_PATH.open("w", newline="", encoding="utf-8-sig") as file:
            writer = csv.DictWriter(file, fieldnames=["word", "meaning", "example", "tag"])
            writer.writeheader()
            writer.writerows(SAMPLE_WORDS)

    if not PROGRESS_PATH.exists():
        write_json(PROGRESS_PATH, {"words": {}})

    if not MEMO_PATH.exists():
        MEMO_PATH.write_text("", encoding="utf-8")

    if not SETTINGS_PATH.exists():
        write_json(SETTINGS_PATH, DEFAULT_SETTINGS)


def read_json(path: Path, fallback: dict) -> dict:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (FileNotFoundError, json.JSONDecodeError):
        return fallback.copy()


def write_json(path: Path, data: dict) -> None:
    path.parent.mkdir(exist_ok=True)
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")


def resolve_app_path(value: str | Path) -> Path:
    path = Path(value)
    if path.is_absolute():
        return path
    return APP_DIR / path


def app_relative_path(path: Path) -> str:
    try:
        return path.resolve().relative_to(APP_DIR.resolve()).as_posix()
    except ValueError:
        return str(path)


def background_supported() -> bool:
    return Image is not None and ImageTk is not None


def copy_background_image(source: Path) -> Path:
    if not source.exists():
        raise FileNotFoundError(f"找不到图片文件：{source}")

    if Image is not None:
        with Image.open(source) as image:
            image.verify()

    ASSETS_DIR.mkdir(exist_ok=True)
    suffix = source.suffix.lower()
    if suffix not in {".png", ".jpg", ".jpeg", ".webp", ".bmp"}:
        suffix = ".png"

    target = ASSETS_DIR / f"background_custom{suffix}"
    if source.resolve() != target.resolve():
        shutil.copy2(source, target)
    return target


def create_cover_image(source: Path, width: int, height: int):
    if not background_supported():
        return None

    width = max(1, width)
    height = max(1, height)

    with Image.open(source) as image:
        image = ImageOps.exif_transpose(image).convert("RGB")
        scale = max(width / image.width, height / image.height)
        resized_size = (max(1, int(image.width * scale)), max(1, int(image.height * scale)))
        resample = getattr(getattr(Image, "Resampling", Image), "LANCZOS")
        image = image.resize(resized_size, resample)

        left = max(0, (image.width - width) // 2)
        top = max(0, (image.height - height) // 2)
        image = image.crop((left, top, left + width, top + height))

        return ImageTk.PhotoImage(image)


def load_settings() -> dict:
    settings = DEFAULT_SETTINGS.copy()
    settings.update(read_json(SETTINGS_PATH, DEFAULT_SETTINGS))
    if "interface_opacity" not in settings and "window_opacity" in settings:
        settings["interface_opacity"] = settings["window_opacity"]
    settings.pop("window_opacity", None)
    return settings


def save_settings(settings: dict) -> None:
    write_json(SETTINGS_PATH, settings)


def load_progress() -> dict:
    progress = read_json(PROGRESS_PATH, {"words": {}})
    if not isinstance(progress.get("words"), dict):
        progress["words"] = {}
    return progress


def save_progress(progress: dict) -> None:
    write_json(PROGRESS_PATH, progress)


def word_key(word: str) -> str:
    return word.strip().casefold()


def load_words() -> list[dict]:
    words: list[dict] = []
    seen: set[str] = set()

    try:
        with WORDS_PATH.open("r", newline="", encoding="utf-8-sig") as file:
            reader = csv.DictReader(file)
            for row in reader:
                word = (row.get("word") or "").strip()
                if not word:
                    continue

                key = word_key(word)
                if key in seen:
                    continue
                seen.add(key)

                words.append(
                    {
                        "key": key,
                        "word": word,
                        "meaning": (row.get("meaning") or "").strip(),
                        "example": (row.get("example") or "").strip(),
                        "tag": (row.get("tag") or "").strip(),
                    }
                )
    except FileNotFoundError:
        ensure_data_files()

    return words


def read_word_rows() -> list[dict]:
    ensure_data_files()
    rows: list[dict] = []
    with WORDS_PATH.open("r", newline="", encoding="utf-8-sig") as file:
        reader = csv.DictReader(file)
        for row in reader:
            rows.append(
                {
                    "word": (row.get("word") or "").strip(),
                    "meaning": (row.get("meaning") or "").strip(),
                    "example": (row.get("example") or "").strip(),
                    "tag": (row.get("tag") or "").strip(),
                }
            )
    return rows


def write_word_rows(rows: list[dict]) -> None:
    DATA_DIR.mkdir(exist_ok=True)
    with WORDS_PATH.open("w", newline="", encoding="utf-8-sig") as file:
        writer = csv.DictWriter(file, fieldnames=["word", "meaning", "example", "tag"])
        writer.writeheader()
        for row in rows:
            writer.writerow(
                {
                    "word": row.get("word", "").strip(),
                    "meaning": row.get("meaning", "").strip(),
                    "example": row.get("example", "").strip(),
                    "tag": row.get("tag", "").strip(),
                }
            )


def add_or_update_word(word: str, meaning: str = "", example: str = "", tag: str = "", update_existing: bool = False) -> str:
    clean_word = word.strip()
    if not clean_word:
        raise ValueError("请填写单词。")

    new_row = {
        "word": clean_word,
        "meaning": meaning.strip(),
        "example": example.strip(),
        "tag": tag.strip(),
    }
    key = word_key(clean_word)
    rows = read_word_rows()

    for index, row in enumerate(rows):
        if word_key(row.get("word", "")) == key:
            if not update_existing:
                return "duplicate"
            rows[index] = new_row
            write_word_rows(rows)
            return "updated"

    rows.append(new_row)
    write_word_rows(rows)
    return "added"


def read_memo() -> str:
    try:
        return MEMO_PATH.read_text(encoding="utf-8")
    except FileNotFoundError:
        return ""


def write_memo(text: str) -> None:
    MEMO_PATH.parent.mkdir(exist_ok=True)
    MEMO_PATH.write_text(text, encoding="utf-8")


def autostart_supported() -> bool:
    return platform.system() == "Windows" and winreg is not None


def build_autostart_command() -> str:
    if getattr(sys, "frozen", False):
        return f'"{Path(sys.executable).resolve()}" --hidden'

    runner = Path(sys.executable).resolve()
    pythonw = runner.with_name("pythonw.exe")
    if pythonw.exists():
        runner = pythonw

    script = Path(__file__).resolve()
    return f'"{runner}" "{script}" --hidden'


def is_autostart_enabled() -> bool:
    if not autostart_supported():
        return False

    try:
        with winreg.OpenKey(winreg.HKEY_CURRENT_USER, RUN_REGISTRY_PATH) as key:
            winreg.QueryValueEx(key, APP_NAME)
            return True
    except OSError:
        return False


def set_autostart_enabled(enabled: bool) -> None:
    if not autostart_supported():
        raise RuntimeError("当前系统不支持 Windows 注册表自启动。")

    with winreg.OpenKey(
        winreg.HKEY_CURRENT_USER,
        RUN_REGISTRY_PATH,
        0,
        winreg.KEY_SET_VALUE,
    ) as key:
        if enabled:
            winreg.SetValueEx(key, APP_NAME, 0, winreg.REG_SZ, build_autostart_command())
        else:
            try:
                winreg.DeleteValue(key, APP_NAME)
            except FileNotFoundError:
                pass


class TrayController:
    def __init__(self, app: "DesktopHelperApp") -> None:
        self.app = app
        self.icon = None
        self.thread: threading.Thread | None = None
        self.available = pystray is not None and Image is not None and ImageDraw is not None

    def start(self) -> bool:
        if not self.available or self.icon is not None:
            return False

        image = self._create_icon_image()
        menu = pystray.Menu(
            pystray.MenuItem("显示主窗口", self._show_window, default=True),
            pystray.MenuItem("隐藏主窗口", self._hide_window),
            pystray.MenuItem("退出程序", self._quit_app),
        )
        self.icon = pystray.Icon(APP_NAME, image, "Desktop Helper", menu)
        self.thread = threading.Thread(target=self.icon.run, daemon=True)
        self.thread.start()
        return True

    def stop(self) -> None:
        if self.icon is not None:
            self.icon.stop()
            self.icon = None

    def _create_icon_image(self):
        image = Image.new("RGB", (64, 64), "#2f6fed")
        draw = ImageDraw.Draw(image)
        draw.rounded_rectangle((10, 8, 54, 56), radius=8, fill="#ffffff")
        draw.rectangle((17, 18, 47, 23), fill="#2f6fed")
        draw.rectangle((17, 30, 41, 35), fill="#2f6fed")
        draw.rectangle((17, 42, 33, 47), fill="#2f6fed")
        return image

    def _show_window(self, _icon=None, _item=None) -> None:
        self.app.root.after(0, self.app.show_window)

    def _hide_window(self, _icon=None, _item=None) -> None:
        self.app.root.after(0, self.app.hide_window)

    def _quit_app(self, _icon=None, _item=None) -> None:
        self.app.root.after(0, self.app.quit_app)


class DesktopHelperApp:
    def __init__(self, root: Tk) -> None:
        ensure_data_files()

        self.root = root
        self.settings = load_settings()
        self.progress = load_progress()
        self.words = load_words()
        self.today_words: list[dict] = []
        self.current_index = 0
        self.is_quitting = False
        self.background_photo = None
        self.background_preview_photo = None
        self.background_preview_after_id = None
        self.background_after_id = None

        self.daily_goal_var = IntVar(value=int(self.settings.get("daily_goal", 10)))
        self.autostart_var = BooleanVar(value=is_autostart_enabled())
        self.start_hidden_var = BooleanVar(value=bool(self.settings.get("start_hidden", False)))
        self.background_enabled_var = BooleanVar(value=bool(self.settings.get("background_enabled", True)))
        self.background_visible_mode_var = BooleanVar(value=bool(self.settings.get("background_visible_mode", False)))
        self.interface_opacity_var = DoubleVar(value=float(self.settings.get("interface_opacity", 100)))
        self.interface_opacity_label_var = StringVar()
        self.search_var = StringVar()
        self.status_var = StringVar()
        self.word_var = StringVar()
        self.meaning_var = StringVar()
        self.example_var = StringVar()
        self.tag_var = StringVar()
        self.counter_var = StringVar()
        self.total_words_var = StringVar()
        self.known_words_var = StringVar()
        self.unknown_words_var = StringVar()
        self.today_studied_var = StringVar()

        self.tray = TrayController(self)

        self._configure_root()
        self._build_ui()
        self.apply_visual_mode(save=False)
        self._bind_events()
        self.reload_words(show_message=False)
        self._load_memo()
        self._refresh_settings_status()

        tray_started = self.tray.start()
        if tray_started:
            self.set_status("托盘已启用。关闭窗口时会隐藏到托盘。")
        else:
            self.set_status("未安装托盘依赖。关闭窗口时会最小化到任务栏。")

        should_start_hidden = "--hidden" in sys.argv or bool(self.settings.get("start_hidden", False))
        if should_start_hidden:
            self.root.after(300, self.hide_window)

    def _configure_root(self) -> None:
        configure_app_style(self.root)
        self.root.title("Desktop Helper")
        geometry = str(self.settings.get("window_geometry", "1040x680"))
        if geometry == DEFAULT_SETTINGS["window_geometry"]:
            geometry = "1040x680"
        self.root.geometry(geometry)
        self.root.minsize(940, 620)

    def _build_ui(self) -> None:
        self.background_label = Label(self.root, bd=0, bg=THEME["window"])
        self.background_label.place(x=0, y=0, relwidth=1, relheight=1)
        self.background_label.lower()

        self.root.columnconfigure(0, weight=1)
        self.root.rowconfigure(1, weight=1)

        self.header_frame = ttk.Frame(self.root, padding=(18, 14), style="Header.TFrame")
        self.header_frame.grid(row=0, column=0, sticky="ew", padx=18, pady=(18, 14))
        self.header_frame.columnconfigure(0, weight=1)

        title = ttk.Label(self.header_frame, text="Desktop Helper", style="Title.TLabel")
        title.grid(row=0, column=0, sticky="w")
        subtitle = ttk.Label(self.header_frame, text="轻量单词学习与备忘录工具，数据全部保存在本地 data 文件夹。", style="Subtitle.TLabel")
        subtitle.grid(row=1, column=0, sticky="w", pady=(4, 0))

        ttk.Button(self.header_frame, text="打开数据文件夹", command=self.open_data_folder).grid(row=0, column=1, rowspan=2, padx=(8, 0))
        ttk.Button(self.header_frame, text="隐藏窗口", command=self.hide_window, style="Accent.TButton").grid(row=0, column=2, rowspan=2, padx=(8, 0))
        ttk.Button(self.header_frame, text="退出", command=self.quit_app, style="Danger.TButton").grid(row=0, column=3, rowspan=2, padx=(8, 0))

        self.notebook = ttk.Notebook(self.root)
        self.notebook.grid(row=1, column=0, sticky="nsew", padx=18)

        self.study_tab = ttk.Frame(self.notebook, padding=16, style="App.TFrame")
        self.words_tab = ttk.Frame(self.notebook, padding=16, style="App.TFrame")
        self.memo_tab = ttk.Frame(self.notebook, padding=16, style="App.TFrame")
        self.background_preview_tab = ttk.Frame(self.notebook, padding=0, style="App.TFrame")
        self.help_tab = ttk.Frame(self.notebook, padding=16, style="App.TFrame")
        self.settings_tab = ttk.Frame(self.notebook, padding=16, style="App.TFrame")

        self.notebook.add(self.study_tab, text="今日学习")
        self.notebook.add(self.words_tab, text="单词列表")
        self.notebook.add(self.memo_tab, text="备忘录")
        self.notebook.add(self.background_preview_tab, text="背景预览")
        self.notebook.add(self.help_tab, text="使用说明")
        self.notebook.add(self.settings_tab, text="设置")

        self._build_study_tab()
        self._build_words_tab()
        self._build_memo_tab()
        self._build_background_preview_tab()
        self._build_help_tab()
        self._build_settings_tab()

        self.status_label = ttk.Label(self.root, textvariable=self.status_var, anchor="w", style="Status.TLabel")
        self.status_label.grid(row=2, column=0, sticky="ew", padx=18, pady=(12, 18))

        self.root.after(100, self.apply_background)

    def _create_metric_card(self, parent: ttk.Frame, column: int, title: str, variable: StringVar) -> None:
        parent.columnconfigure(column, weight=1, uniform="metrics")
        card = ttk.Frame(parent, padding=(14, 12), style="Card.TFrame")
        card.grid(row=0, column=column, sticky="ew", padx=(0 if column == 0 else 8, 0))
        ttk.Label(card, text=title, style="MetricTitle.TLabel").grid(row=0, column=0, sticky="w")
        ttk.Label(card, textvariable=variable, style="MetricValue.TLabel").grid(row=1, column=0, sticky="w", pady=(4, 0))

    def _build_study_tab(self) -> None:
        self.study_tab.columnconfigure(0, weight=1)
        self.study_tab.rowconfigure(2, weight=1)

        top = ttk.Frame(self.study_tab, padding=(16, 12), style="Surface.TFrame")
        top.grid(row=0, column=0, sticky="ew")
        top.columnconfigure(0, weight=1)

        ttk.Label(top, text="今日学习", style="Title.TLabel").grid(row=0, column=0, sticky="w")
        ttk.Label(top, textvariable=self.counter_var, style="Subtitle.TLabel").grid(row=1, column=0, sticky="w", pady=(4, 0))
        ttk.Button(top, text="重新加载单词", command=self.reload_words, style="Accent.TButton").grid(row=0, column=1, rowspan=2, padx=(12, 0))

        metrics = ttk.Frame(self.study_tab, style="App.TFrame")
        metrics.grid(row=1, column=0, sticky="ew", pady=(12, 12))
        self._create_metric_card(metrics, 0, "总词数", self.total_words_var)
        self._create_metric_card(metrics, 1, "已认识", self.known_words_var)
        self._create_metric_card(metrics, 2, "不认识", self.unknown_words_var)
        self._create_metric_card(metrics, 3, "今日已学", self.today_studied_var)

        card = ttk.Frame(self.study_tab, padding=24, style="Card.TFrame")
        card.grid(row=2, column=0, sticky="nsew", pady=(0, 12))
        card.columnconfigure(0, weight=1)
        card.rowconfigure(4, weight=1)

        ttk.Label(card, text="当前单词", style="CardMuted.TLabel").grid(row=0, column=0, sticky="w")
        ttk.Label(card, textvariable=self.word_var, style="Word.TLabel").grid(row=1, column=0, sticky="w", pady=(10, 0))
        ttk.Label(card, textvariable=self.meaning_var, style="Meaning.TLabel").grid(row=2, column=0, sticky="w", pady=(14, 0))
        ttk.Label(card, textvariable=self.example_var, wraplength=850, style="CardMuted.TLabel").grid(row=3, column=0, sticky="w", pady=(14, 0))
        ttk.Label(card, textvariable=self.tag_var, style="CardMuted.TLabel").grid(row=4, column=0, sticky="nw", pady=(12, 0))

        actions = ttk.Frame(self.study_tab, style="App.TFrame")
        actions.grid(row=3, column=0, sticky="nw")
        ttk.Button(actions, text="不认识", command=lambda: self.mark_current_word("unknown"), style="Danger.TButton").pack(side=LEFT, padx=(0, 8))
        ttk.Button(actions, text="认识", command=lambda: self.mark_current_word("known"), style="Success.TButton").pack(side=LEFT, padx=(0, 8))
        ttk.Button(actions, text="上一个", command=self.previous_word).pack(side=LEFT, padx=(0, 8))
        ttk.Button(actions, text="下一个", command=self.next_word).pack(side=LEFT)

    def _build_words_tab(self) -> None:
        self.words_tab.columnconfigure(0, weight=1)
        self.words_tab.rowconfigure(1, weight=1)

        controls = ttk.Frame(self.words_tab, padding=(14, 12), style="Surface.TFrame")
        controls.grid(row=0, column=0, columnspan=2, sticky="ew", pady=(0, 12))
        controls.columnconfigure(1, weight=1)

        ttk.Label(controls, text="搜索", style="Surface.TLabel").grid(row=0, column=0, sticky="w")
        search = ttk.Entry(controls, textvariable=self.search_var)
        search.grid(row=0, column=1, sticky="ew", padx=(10, 10))
        ttk.Button(controls, text="添加单词", command=self.open_add_word_dialog, style="Accent.TButton").grid(row=0, column=2, padx=(0, 8))
        ttk.Button(controls, text="导入 CSV", command=self.import_words_csv).grid(row=0, column=3, padx=(0, 8))
        ttk.Button(controls, text="重新加载", command=self.reload_words).grid(row=0, column=4)

        columns = ("word", "meaning", "tag", "status", "last_studied")
        self.words_tree = ttk.Treeview(self.words_tab, columns=columns, show="headings", height=14)
        self.words_tree.heading("word", text="单词")
        self.words_tree.heading("meaning", text="释义")
        self.words_tree.heading("tag", text="标签")
        self.words_tree.heading("status", text="状态")
        self.words_tree.heading("last_studied", text="上次学习")
        self.words_tree.column("word", width=150, anchor="w")
        self.words_tree.column("meaning", width=260, anchor="w")
        self.words_tree.column("tag", width=100, anchor="w")
        self.words_tree.column("status", width=90, anchor="w")
        self.words_tree.column("last_studied", width=140, anchor="w")
        self.words_tree.grid(row=1, column=0, sticky="nsew")
        self.words_tree.tag_configure("known", foreground=THEME["success"])
        self.words_tree.tag_configure("unknown", foreground=THEME["danger"])
        self.words_tree.tag_configure("new", foreground=THEME["muted"])

        scrollbar = ttk.Scrollbar(self.words_tab, orient="vertical", command=self.words_tree.yview)
        scrollbar.grid(row=1, column=1, sticky="ns")
        self.words_tree.configure(yscrollcommand=scrollbar.set)

        hint = ttk.Label(self.words_tab, text=f"单词源文件：{WORDS_PATH}", style="Muted.TLabel")
        hint.grid(row=2, column=0, sticky="w", pady=(8, 0))

    def _build_memo_tab(self) -> None:
        self.memo_tab.columnconfigure(0, weight=1)
        self.memo_tab.rowconfigure(0, weight=1)

        self.memo_text = Text(
            self.memo_tab,
            wrap="word",
            undo=True,
            font=("Microsoft YaHei UI", 11),
            bg=THEME["card"],
            fg=THEME["text"],
            insertbackground=THEME["text"],
            relief="flat",
            bd=0,
            padx=16,
            pady=14,
            highlightthickness=1,
            highlightbackground=THEME["line"],
            highlightcolor=THEME["primary"],
        )
        self.memo_text.grid(row=0, column=0, sticky="nsew")

        scrollbar = ttk.Scrollbar(self.memo_tab, orient="vertical", command=self.memo_text.yview)
        scrollbar.grid(row=0, column=1, sticky="ns")
        self.memo_text.configure(yscrollcommand=scrollbar.set)

        actions = ttk.Frame(self.memo_tab, style="App.TFrame")
        actions.grid(row=1, column=0, sticky="ew", pady=(8, 0))
        ttk.Button(actions, text="保存备忘录", command=self.save_memo, style="Accent.TButton").pack(side=LEFT)
        ttk.Label(actions, text=f"保存位置：{MEMO_PATH}", style="Muted.TLabel").pack(side=LEFT, padx=(12, 0))

    def _build_background_preview_tab(self) -> None:
        self.background_preview_tab.columnconfigure(0, weight=1)
        self.background_preview_tab.rowconfigure(0, weight=1)

        self.background_preview_label = Label(
            self.background_preview_tab,
            bd=0,
            bg="#111827",
            anchor="center",
            text="未找到背景图",
            fg="#ffffff",
            font=("Microsoft YaHei UI", 13),
        )
        self.background_preview_label.grid(row=0, column=0, sticky="nsew")

        actions = ttk.Frame(self.background_preview_tab, padding=(12, 10), style="Surface.TFrame")
        actions.grid(row=1, column=0, sticky="ew")
        actions.columnconfigure(3, weight=1)
        ttk.Button(actions, text="更换背景图", command=self.choose_background_image, style="Accent.TButton").grid(row=0, column=0, sticky="w", padx=(0, 8))
        ttk.Button(actions, text="恢复默认背景", command=self.reset_background_image).grid(row=0, column=1, sticky="w", padx=(0, 8))
        ttk.Button(actions, text="打开图片文件", command=self.open_background_image).grid(row=0, column=2, sticky="w", padx=(0, 12))
        self.background_preview_path_var = StringVar(value=f"当前背景：{self.get_background_path()}")
        ttk.Label(actions, textvariable=self.background_preview_path_var, style="Surface.TLabel").grid(row=0, column=3, sticky="e")

        self.background_preview_label.bind("<Configure>", lambda _event: self.schedule_background_preview_refresh())
        self.root.after(150, self.apply_background_preview)

    def _build_help_tab(self) -> None:
        self.help_tab.columnconfigure(0, weight=1)
        self.help_tab.rowconfigure(0, weight=1)

        self.help_text = Text(
            self.help_tab,
            wrap="word",
            font=("Microsoft YaHei UI", 11),
            height=18,
            bg=THEME["card"],
            fg=THEME["text"],
            relief="flat",
            bd=0,
            padx=16,
            pady=14,
            highlightthickness=1,
            highlightbackground=THEME["line"],
            highlightcolor=THEME["primary"],
        )
        self.help_text.grid(row=0, column=0, sticky="nsew")
        self.help_text.insert("1.0", USAGE_TEXT)
        self.help_text.configure(state="disabled")

        scrollbar = ttk.Scrollbar(self.help_tab, orient="vertical", command=self.help_text.yview)
        scrollbar.grid(row=0, column=1, sticky="ns")
        self.help_text.configure(yscrollcommand=scrollbar.set)

        actions = ttk.Frame(self.help_tab, style="App.TFrame")
        actions.grid(row=1, column=0, sticky="ew", pady=(8, 0))
        ttk.Button(actions, text="打开 README", command=self.open_readme, style="Accent.TButton").pack(side=LEFT)
        ttk.Label(actions, text=f"文档位置：{README_PATH}", style="Muted.TLabel").pack(side=LEFT, padx=(12, 0))

    def _build_settings_tab(self) -> None:
        self.settings_tab.columnconfigure(0, weight=1)

        panel = ttk.Frame(self.settings_tab, padding=(16, 14), style="Surface.TFrame")
        panel.grid(row=0, column=0, sticky="ew")
        panel.columnconfigure(1, weight=1)

        ttk.Label(panel, text="设置", style="Title.TLabel").grid(row=0, column=0, columnspan=2, sticky="w")
        ttk.Label(panel, text="控制每日学习数量、启动行为和 Windows 自启动。", style="Subtitle.TLabel").grid(
            row=1, column=0, columnspan=2, sticky="w", pady=(4, 14)
        )

        ttk.Label(panel, text="每日学习数量", style="Surface.TLabel").grid(row=2, column=0, sticky="w", pady=(0, 10))
        ttk.Spinbox(
            panel,
            from_=1,
            to=200,
            textvariable=self.daily_goal_var,
            width=8,
            command=self.save_settings_from_ui,
        ).grid(row=2, column=1, sticky="w", pady=(0, 10))

        autostart = ttk.Checkbutton(
            panel,
            text="开机自启动",
            variable=self.autostart_var,
            command=self.toggle_autostart,
            style="Surface.TCheckbutton",
        )
        autostart.grid(row=3, column=0, columnspan=2, sticky="w", pady=(0, 10))
        if not autostart_supported():
            autostart.state(["disabled"])

        ttk.Checkbutton(
            panel,
            text="启动后自动隐藏",
            variable=self.start_hidden_var,
            command=self.save_settings_from_ui,
            style="Surface.TCheckbutton",
        ).grid(row=4, column=0, columnspan=2, sticky="w", pady=(0, 10))

        ttk.Checkbutton(
            panel,
            text="启用背景图",
            variable=self.background_enabled_var,
            command=self.toggle_background,
            style="Surface.TCheckbutton",
        ).grid(row=5, column=0, columnspan=2, sticky="w", pady=(0, 10))

        ttk.Checkbutton(
            panel,
            text="背景可见模式",
            variable=self.background_visible_mode_var,
            command=self.toggle_visual_mode,
            style="Surface.TCheckbutton",
        ).grid(row=6, column=0, columnspan=2, sticky="w", pady=(0, 10))

        ttk.Label(panel, text="界面淡化程度", style="Surface.TLabel").grid(row=7, column=0, sticky="w", pady=(0, 10))
        fade_row = ttk.Frame(panel, style="Surface.TFrame")
        fade_row.grid(row=7, column=1, sticky="ew", pady=(0, 10))
        fade_row.columnconfigure(0, weight=1)
        ttk.Scale(
            fade_row,
            from_=70,
            to=100,
            variable=self.interface_opacity_var,
            command=lambda _value: self.apply_visual_mode(),
        ).grid(row=0, column=0, sticky="ew", padx=(0, 10))
        ttk.Label(fade_row, textvariable=self.interface_opacity_label_var, width=5, style="Surface.TLabel").grid(row=0, column=1, sticky="e")

        background_actions = ttk.Frame(panel, style="Surface.TFrame")
        background_actions.grid(row=8, column=0, columnspan=2, sticky="w", pady=(0, 12))
        ttk.Button(background_actions, text="更换背景图", command=self.choose_background_image).pack(side=LEFT, padx=(0, 8))
        ttk.Button(background_actions, text="恢复默认背景", command=self.reset_background_image).pack(side=LEFT)

        ttk.Button(panel, text="保存设置", command=self.save_settings_from_ui, style="Accent.TButton").grid(row=9, column=0, sticky="w", pady=(4, 0))

        info_card = ttk.Frame(self.settings_tab, padding=(16, 14), style="Card.TFrame")
        info_card.grid(row=1, column=0, sticky="ew", pady=(12, 0))
        info_card.columnconfigure(0, weight=1)
        ttk.Label(info_card, text="运行状态", style="MetricTitle.TLabel").grid(row=0, column=0, sticky="w")
        self.settings_info = ttk.Label(info_card, justify=LEFT, wraplength=860, style="CardMuted.TLabel")
        self.settings_info.grid(row=1, column=0, sticky="ew", pady=(8, 0))

    def _safe_interface_opacity(self) -> int:
        try:
            value = int(round(float(self.interface_opacity_var.get())))
        except Exception:
            value = int(self.settings.get("interface_opacity", 100))
        value = clamp_int(value, 70, 100)
        self.interface_opacity_var.set(value)
        self.interface_opacity_label_var.set(f"{value}%")
        return value

    def apply_visual_mode(self, save: bool = True) -> None:
        visible_mode = bool(self.background_visible_mode_var.get())
        interface_opacity = self._safe_interface_opacity()

        configure_app_style(self.root, visible_mode=visible_mode, interface_opacity=interface_opacity)

        outer_pad = 34 if visible_mode else 18
        header_pady = (28, 16) if visible_mode else (18, 14)
        status_pady = (12, 28) if visible_mode else (12, 18)
        tab_padding = 10 if visible_mode else 16

        if hasattr(self, "header_frame"):
            self.header_frame.grid_configure(padx=outer_pad, pady=header_pady)
        if hasattr(self, "notebook"):
            self.notebook.grid_configure(padx=outer_pad)
        if hasattr(self, "status_label"):
            self.status_label.grid_configure(padx=outer_pad, pady=status_pady)

        for tab_name in ("study_tab", "words_tab", "memo_tab", "help_tab", "settings_tab"):
            tab = getattr(self, tab_name, None)
            if tab is not None:
                tab.configure(padding=tab_padding)
        if hasattr(self, "background_preview_tab"):
            self.background_preview_tab.configure(padding=0)

        text_bg = THEME["card_visible"] if visible_mode else THEME["card"]
        for text_widget_name in ("memo_text", "help_text"):
            widget = getattr(self, text_widget_name, None)
            if widget is not None:
                widget.configure(bg=text_bg)

        try:
            self.root.attributes("-alpha", 1.0)
        except Exception:
            pass

        self.settings["background_visible_mode"] = visible_mode
        self.settings["interface_opacity"] = interface_opacity
        if save:
            save_settings(self.settings)
            self._refresh_settings_status()

    def apply_background_preview(self) -> None:
        self.background_preview_after_id = None
        if not hasattr(self, "background_preview_label"):
            return

        background_path = self.get_background_path()
        if hasattr(self, "background_preview_path_var"):
            self.background_preview_path_var.set(f"当前背景：{background_path}")

        if not background_supported() or not background_path.exists():
            self.background_preview_photo = None
            self.background_preview_label.configure(image="", text="未找到背景图", bg="#111827")
            return

        width = max(1, self.background_preview_label.winfo_width())
        height = max(1, self.background_preview_label.winfo_height())
        if width < 50 or height < 50:
            return

        try:
            photo = create_cover_image(background_path, width, height)
        except Exception as error:
            self.background_preview_photo = None
            self.background_preview_label.configure(image="", text=f"背景图加载失败：{error}", bg="#111827")
            return

        self.background_preview_photo = photo
        self.background_preview_label.configure(image=photo, text="")

    def schedule_background_preview_refresh(self) -> None:
        if not hasattr(self, "background_preview_label"):
            return
        if self.background_preview_after_id is not None:
            try:
                self.root.after_cancel(self.background_preview_after_id)
            except Exception:
                pass
        self.background_preview_after_id = self.root.after(120, self.apply_background_preview)

    def toggle_visual_mode(self) -> None:
        self.apply_visual_mode()
        state = "开启" if self.background_visible_mode_var.get() else "关闭"
        self.set_status(f"背景可见模式已{state}。")

    def get_background_path(self) -> Path:
        return resolve_app_path(self.settings.get("background_image", DEFAULT_SETTINGS["background_image"]))

    def apply_background(self) -> None:
        self.background_after_id = None
        if not hasattr(self, "background_label"):
            return

        if not self.background_enabled_var.get():
            self.background_label.configure(image="", bg=THEME["window"])
            self.background_photo = None
            return

        if not background_supported():
            self.background_label.configure(image="", bg=THEME["window"])
            self.background_photo = None
            return

        background_path = self.get_background_path()
        if not background_path.exists():
            self.background_label.configure(image="", bg=THEME["window"])
            self.background_photo = None
            return

        width = max(1, self.root.winfo_width())
        height = max(1, self.root.winfo_height())
        if width < 50 or height < 50:
            return

        try:
            photo = create_cover_image(background_path, width, height)
        except Exception as error:
            self.background_label.configure(image="", bg=THEME["window"])
            self.background_photo = None
            self.set_status(f"背景图加载失败：{error}")
            return

        if photo is not None:
            self.background_photo = photo
            self.background_label.configure(image=photo)
            self.background_label.lower()

    def schedule_background_refresh(self) -> None:
        if not hasattr(self, "background_label"):
            return
        if self.background_after_id is not None:
            try:
                self.root.after_cancel(self.background_after_id)
            except Exception:
                pass
        self.background_after_id = self.root.after(120, self.apply_background)

    def toggle_background(self) -> None:
        self.settings["background_enabled"] = bool(self.background_enabled_var.get())
        save_settings(self.settings)
        self.apply_background()
        self._refresh_settings_status()
        state = "启用" if self.background_enabled_var.get() else "关闭"
        self.set_status(f"背景图已{state}。")

    def choose_background_image(self) -> None:
        if not background_supported():
            messagebox.showwarning("缺少依赖", "更换背景图需要安装 Pillow。请先执行：pip install -r requirements.txt")
            return

        path = filedialog.askopenfilename(
            title="选择背景图片",
            filetypes=[
                ("图片文件", "*.png;*.jpg;*.jpeg;*.webp;*.bmp"),
                ("所有文件", "*.*"),
            ],
        )
        if not path:
            return

        try:
            target = copy_background_image(Path(path))
        except Exception as error:
            messagebox.showerror("更换背景失败", str(error))
            return

        self.settings["background_image"] = app_relative_path(target)
        self.settings["background_enabled"] = True
        self.background_enabled_var.set(True)
        save_settings(self.settings)
        self.apply_background()
        self.apply_background_preview()
        self._refresh_settings_status()
        self.set_status(f"已更换背景图：{target.name}")

    def reset_background_image(self) -> None:
        if not DEFAULT_BACKGROUND_PATH.exists():
            messagebox.showwarning("默认背景不存在", f"没有找到默认背景：{DEFAULT_BACKGROUND_PATH}")
            return

        self.settings["background_image"] = app_relative_path(DEFAULT_BACKGROUND_PATH)
        self.settings["background_enabled"] = True
        self.background_enabled_var.set(True)
        save_settings(self.settings)
        self.apply_background()
        self.apply_background_preview()
        self._refresh_settings_status()
        self.set_status("已恢复默认背景图。")

    def _bind_events(self) -> None:
        self.root.protocol("WM_DELETE_WINDOW", self.on_close)
        self.root.bind("<Unmap>", self._save_geometry_on_change)
        self.root.bind("<Configure>", self._save_geometry_on_change)
        self.root.bind("<Control-n>", lambda _event: self.open_add_word_dialog())
        self.search_var.trace_add("write", lambda *_args: self.refresh_words_tree())

    def _save_geometry_on_change(self, _event=None) -> None:
        if _event is not None and _event.widget is self.root:
            self.schedule_background_refresh()

        if self.root.state() == "normal":
            self.settings["window_geometry"] = self.root.geometry()

    def reload_words(self, show_message: bool = True) -> None:
        self.words = load_words()
        self._build_today_words()
        self.current_index = 0
        self.refresh_study_card()
        self.refresh_words_tree()
        if show_message:
            self.set_status(f"已重新加载 {len(self.words)} 个单词。")

    def _build_today_words(self) -> None:
        goal = self._safe_daily_goal()
        not_known = [word for word in self.words if self._word_status(word["key"]) != "known"]
        source = not_known if not_known else self.words
        self.today_words = source[:goal]

    def _safe_daily_goal(self) -> int:
        try:
            value = int(self.daily_goal_var.get())
        except Exception:
            value = int(self.settings.get("daily_goal", 10))
        return max(1, min(value, 200))

    def _refresh_metrics(self, studied_today: int) -> None:
        total = len(self.words)
        known = sum(1 for word in self.words if self._word_status(word["key"]) == "known")
        unknown = sum(1 for word in self.words if self._word_status(word["key"]) == "unknown")

        self.total_words_var.set(str(total))
        self.known_words_var.set(str(known))
        self.unknown_words_var.set(str(unknown))
        self.today_studied_var.set(str(studied_today))

    def refresh_study_card(self) -> None:
        studied_today = sum(
            1
            for item in self.progress.get("words", {}).values()
            if item.get("last_studied") == today_text()
        )
        self._refresh_metrics(studied_today)

        if not self.today_words:
            self.word_var.set("暂无单词")
            self.meaning_var.set(f"请在 {WORDS_PATH} 中添加单词，或点击“导入 CSV”。")
            self.example_var.set("")
            self.tag_var.set("")
            self.counter_var.set(f"今日已学习：{studied_today} / 目标 {self._safe_daily_goal()}")
            return

        self.current_index = max(0, min(self.current_index, len(self.today_words) - 1))
        word = self.today_words[self.current_index]

        self.word_var.set(word["word"])
        self.meaning_var.set(word["meaning"] or "未填写释义")
        self.example_var.set(word["example"] or "未填写例句")
        self.tag_var.set(f"标签：{word['tag'] or '无'}")
        self.counter_var.set(
            f"今日进度：{self.current_index + 1} / {len(self.today_words)}    "
            f"今日已学习：{studied_today} / 目标 {self._safe_daily_goal()}"
        )

    def refresh_words_tree(self) -> None:
        for item_id in self.words_tree.get_children():
            self.words_tree.delete(item_id)

        keyword = self.search_var.get().strip().casefold()
        for word in self.words:
            haystack = " ".join([word["word"], word["meaning"], word["tag"]]).casefold()
            if keyword and keyword not in haystack:
                continue

            item = self.progress.get("words", {}).get(word["key"], {})
            status_key = item.get("status", "new")
            status = self._status_label(status_key)
            last_studied = item.get("last_studied", "")
            self.words_tree.insert(
                "",
                END,
                values=(word["word"], word["meaning"], word["tag"], status, last_studied),
                tags=(status_key if status_key in {"known", "unknown", "new"} else "new",),
            )

    def open_add_word_dialog(self) -> None:
        dialog = Toplevel(self.root)
        dialog.title("添加单词")
        dialog.geometry("520x460")
        dialog.minsize(500, 430)
        dialog.configure(bg=THEME["window"])
        dialog.transient(self.root)
        dialog.grab_set()

        word_var = StringVar()
        meaning_var = StringVar()
        tag_var = StringVar()
        form_status_var = StringVar(value="单词为必填；释义、例句和标签可以留空。")

        container = ttk.Frame(dialog, padding=18, style="App.TFrame")
        container.grid(row=0, column=0, sticky="nsew")
        container.columnconfigure(0, weight=1)
        container.rowconfigure(1, weight=1)
        dialog.columnconfigure(0, weight=1)
        dialog.rowconfigure(0, weight=1)

        header = ttk.Frame(container, padding=(16, 14), style="Surface.TFrame")
        header.grid(row=0, column=0, sticky="ew", pady=(0, 12))
        header.columnconfigure(0, weight=1)
        ttk.Label(header, text="添加单词", style="Title.TLabel").grid(row=0, column=0, sticky="w")
        ttk.Label(header, text="手动新增一个单词，保存后会写入 data/words.csv。", style="Subtitle.TLabel").grid(
            row=1, column=0, sticky="w", pady=(4, 0)
        )

        form = ttk.Frame(container, padding=(16, 14), style="Card.TFrame")
        form.grid(row=1, column=0, sticky="nsew")
        form.columnconfigure(1, weight=1)
        form.rowconfigure(3, weight=1)

        ttk.Label(form, text="单词 *", style="Card.TLabel").grid(row=0, column=0, sticky="w", pady=(0, 10))
        word_entry = ttk.Entry(form, textvariable=word_var)
        word_entry.grid(row=0, column=1, sticky="ew", pady=(0, 10))

        ttk.Label(form, text="释义", style="Card.TLabel").grid(row=1, column=0, sticky="w", pady=(0, 10))
        meaning_entry = ttk.Entry(form, textvariable=meaning_var)
        meaning_entry.grid(row=1, column=1, sticky="ew", pady=(0, 10))

        ttk.Label(form, text="标签", style="Card.TLabel").grid(row=2, column=0, sticky="w", pady=(0, 10))
        tag_entry = ttk.Entry(form, textvariable=tag_var)
        tag_entry.grid(row=2, column=1, sticky="ew", pady=(0, 10))

        ttk.Label(form, text="例句", style="Card.TLabel").grid(row=3, column=0, sticky="nw")
        example_text = Text(
            form,
            wrap="word",
            height=5,
            font=("Microsoft YaHei UI", 10),
            bg=THEME["input"],
            fg=THEME["text"],
            insertbackground=THEME["text"],
            relief="flat",
            bd=0,
            padx=10,
            pady=8,
            highlightthickness=1,
            highlightbackground=THEME["line"],
            highlightcolor=THEME["primary"],
        )
        example_text.grid(row=3, column=1, sticky="nsew")

        ttk.Label(form, textvariable=form_status_var, style="CardMuted.TLabel", wraplength=410).grid(
            row=4, column=0, columnspan=2, sticky="w", pady=(12, 0)
        )

        actions = ttk.Frame(container, style="App.TFrame")
        actions.grid(row=2, column=0, sticky="ew", pady=(12, 0))

        def clear_form() -> None:
            word_var.set("")
            meaning_var.set("")
            tag_var.set("")
            example_text.delete("1.0", END)
            form_status_var.set("已保存。可以继续添加下一个单词。")
            word_entry.focus_set()

        def save_word(close_after_save: bool) -> None:
            word = word_var.get().strip()
            meaning = meaning_var.get().strip()
            tag = tag_var.get().strip()
            example = example_text.get("1.0", "end-1c").strip()

            if not word:
                form_status_var.set("请先填写单词。")
                word_entry.focus_set()
                return

            try:
                result = add_or_update_word(word, meaning, example, tag)
                if result == "duplicate":
                    should_update = messagebox.askyesno(
                        "单词已存在",
                        f"“{word}” 已经在单词表中。\n\n是否用当前填写的内容更新已有单词？",
                        parent=dialog,
                    )
                    if not should_update:
                        form_status_var.set("未保存：单词已存在。")
                        word_entry.focus_set()
                        word_entry.selection_range(0, END)
                        return
                    result = add_or_update_word(word, meaning, example, tag, update_existing=True)
            except Exception as error:
                form_status_var.set(str(error))
                return

            self.search_var.set(word)
            self.reload_words(show_message=False)
            action = "更新" if result == "updated" else "添加"
            self.set_status(f"已{action}单词：{word}")

            if close_after_save:
                dialog.destroy()
            else:
                clear_form()

        ttk.Button(actions, text="取消", command=dialog.destroy).pack(side=RIGHT)
        ttk.Button(actions, text="保存", command=lambda: save_word(True), style="Accent.TButton").pack(side=RIGHT, padx=(0, 8))
        ttk.Button(actions, text="保存并继续添加", command=lambda: save_word(False)).pack(side=RIGHT, padx=(0, 8))

        word_entry.focus_set()
        dialog.bind("<Escape>", lambda _event: dialog.destroy())
        dialog.bind("<Control-Return>", lambda _event: save_word(True))
        dialog.bind("<Control-s>", lambda _event: save_word(True))

    def _status_label(self, status: str) -> str:
        labels = {
            "known": "认识",
            "unknown": "不认识",
            "new": "未学习",
        }
        return labels.get(status, "未学习")

    def _word_status(self, key: str) -> str:
        return self.progress.get("words", {}).get(key, {}).get("status", "new")

    def mark_current_word(self, status: str) -> None:
        if not self.today_words:
            return

        word = self.today_words[self.current_index]
        item = self.progress.setdefault("words", {}).setdefault(word["key"], {})
        item["word"] = word["word"]
        item["status"] = status
        item["last_studied"] = today_text()
        item["updated_at"] = now_text()

        if status == "known":
            item["known_count"] = int(item.get("known_count", 0)) + 1
            self.set_status(f"已标记为认识：{word['word']}")
        else:
            item["unknown_count"] = int(item.get("unknown_count", 0)) + 1
            self.set_status(f"已标记为不认识：{word['word']}")

        save_progress(self.progress)
        self.refresh_words_tree()
        self.next_word()

    def previous_word(self) -> None:
        if not self.today_words:
            return
        self.current_index = max(0, self.current_index - 1)
        self.refresh_study_card()

    def next_word(self) -> None:
        if not self.today_words:
            return
        if self.current_index < len(self.today_words) - 1:
            self.current_index += 1
        self.refresh_study_card()

    def import_words_csv(self) -> None:
        path = filedialog.askopenfilename(
            title="选择单词 CSV 文件",
            filetypes=[("CSV 文件", "*.csv"), ("所有文件", "*.*")],
        )
        if not path:
            return

        source = Path(path)
        try:
            text = source.read_text(encoding="utf-8-sig")
        except UnicodeDecodeError:
            text = source.read_text(encoding="gbk")

        WORDS_PATH.write_text(text, encoding="utf-8-sig")
        self.reload_words(show_message=False)
        self.set_status(f"已导入单词文件：{source.name}")

    def _load_memo(self) -> None:
        self.memo_text.delete("1.0", END)
        self.memo_text.insert("1.0", read_memo())

    def save_memo(self) -> None:
        write_memo(self.memo_text.get("1.0", "end-1c"))
        self.set_status("备忘录已保存。")

    def save_settings_from_ui(self) -> None:
        self.settings["daily_goal"] = self._safe_daily_goal()
        self.settings["start_hidden"] = bool(self.start_hidden_var.get())
        self.settings["autostart"] = bool(self.autostart_var.get())
        self.settings["background_enabled"] = bool(self.background_enabled_var.get())
        self.settings["background_visible_mode"] = bool(self.background_visible_mode_var.get())
        self.settings["interface_opacity"] = self._safe_interface_opacity()
        self.settings.pop("window_opacity", None)
        self.settings.setdefault("background_image", DEFAULT_SETTINGS["background_image"])
        save_settings(self.settings)
        self._build_today_words()
        self.refresh_study_card()
        self.apply_visual_mode(save=False)
        self.schedule_background_refresh()
        self._refresh_settings_status()
        self.set_status("设置已保存。")

    def toggle_autostart(self) -> None:
        enabled = bool(self.autostart_var.get())
        try:
            set_autostart_enabled(enabled)
        except Exception as error:
            self.autostart_var.set(is_autostart_enabled())
            messagebox.showerror("自启动设置失败", str(error))
            self._refresh_settings_status()
            return

        self.save_settings_from_ui()
        state = "开启" if enabled else "关闭"
        self.set_status(f"开机自启动已{state}。")

    def _refresh_settings_status(self) -> None:
        autostart_text = "支持" if autostart_supported() else "当前系统不支持"
        tray_text = "已启用" if self.tray.available else "未启用，需安装 pystray 和 Pillow"
        background_text = "已启用" if self.background_enabled_var.get() and background_supported() else "未启用"
        if self.background_enabled_var.get() and not background_supported():
            background_text = "未启用，需安装 Pillow"
        visible_mode_text = "开启" if self.background_visible_mode_var.get() else "关闭"
        interface_opacity_text = f"{self._safe_interface_opacity()}%"
        command = build_autostart_command() if autostart_supported() else "不可用"

        self.settings_info.configure(
            text=(
                f"数据目录：{DATA_DIR}\n"
                f"单词文件：{WORDS_PATH}\n"
                f"背景状态：{background_text}\n"
                f"背景文件：{self.get_background_path()}\n"
                f"背景可见模式：{visible_mode_text}\n"
                f"界面淡化程度：{interface_opacity_text}\n"
                f"托盘状态：{tray_text}\n"
                f"Windows 自启动：{autostart_text}\n"
                f"自启动命令：{command}"
            )
        )

    def open_data_folder(self) -> None:
        DATA_DIR.mkdir(exist_ok=True)
        try:
            if platform.system() == "Windows":
                os.startfile(DATA_DIR)
            else:
                messagebox.showinfo("数据目录", str(DATA_DIR))
        except OSError as error:
            messagebox.showerror("打开失败", str(error))

    def open_readme(self) -> None:
        try:
            if platform.system() == "Windows":
                os.startfile(README_PATH)
            else:
                messagebox.showinfo("README", str(README_PATH))
        except OSError as error:
            messagebox.showerror("打开失败", str(error))

    def open_background_image(self) -> None:
        background_path = self.get_background_path()
        if not background_path.exists():
            messagebox.showwarning("背景图不存在", f"没有找到背景图：{background_path}")
            return
        try:
            if platform.system() == "Windows":
                os.startfile(background_path)
            else:
                messagebox.showinfo("背景图", str(background_path))
        except OSError as error:
            messagebox.showerror("打开失败", str(error))

    def on_close(self) -> None:
        if self.is_quitting:
            self.quit_app()
            return
        self.hide_window()

    def hide_window(self) -> None:
        self.save_memo()
        self.save_settings_from_ui()

        if self.tray.available:
            self.root.withdraw()
            self.set_status("窗口已隐藏到托盘。")
        else:
            self.root.iconify()
            self.set_status("未安装托盘依赖，已最小化到任务栏。")

    def show_window(self) -> None:
        self.root.deiconify()
        self.root.lift()
        self.root.focus_force()
        self.set_status("窗口已显示。")

    def quit_app(self) -> None:
        self.is_quitting = True
        self.save_memo()
        self.save_settings_from_ui()
        self.tray.stop()
        self.root.destroy()

    def set_status(self, text: str) -> None:
        self.status_var.set(text)


def main() -> None:
    root = Tk()
    DesktopHelperApp(root)
    root.mainloop()


if __name__ == "__main__":
    main()
