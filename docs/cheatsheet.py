#!/usr/bin/env python3
"""Generate the one-page Toe cheatsheet.

Edit the section lists below, then run `make cheatsheet` from the repository
root. ReportLab is the only dependency.
"""

from pathlib import Path
import sys

try:
    from reportlab.lib.colors import HexColor
    from reportlab.lib.pagesizes import A4, landscape
    from reportlab.pdfbase.pdfmetrics import stringWidth
    from reportlab.pdfgen import canvas
except ImportError:
    sys.exit("reportlab is required: python3 -m pip install reportlab")


OUTPUT = Path(__file__).parent / "static/downloads/toe-cheatsheet.pdf"

ESSENTIALS = [
    ("i", "insert"),
    ("Esc", "normal"),
    ("v", "select"),
    ("Space", "leader"),
    (":", "command"),
    (":w", "save"),
    (":q", "quit"),
    ("Space+?", "palette"),
]

SECTIONS = [
    (
        "01  NAVIGATION & MOVEMENT",
        [
            ("h  j  k  l", "left / down / up / right"),
            ("w  b  e", "next / previous / end of word"),
            ("W  B  E", "same, using long words"),
            ("f<char> / t<char>", "find / move till character"),
            ("F<char> / T<char>", "find / till backward"),
            ("gg / G", "file start / last line"),
            ("<n>gg or <n>G", "go to line number"),
            ("gs / ge", "first non-blank / last line"),
            ("Home / End", "line start / line end"),
            ("Ctrl+f / Ctrl+b", "page down / up"),
            ("Ctrl+d / Ctrl+u", "half-page down / up"),
            ("Shift+Tab / Tab", "jumplist back / forward"),
            ("gn / gp", "next / previous buffer"),
            ("[p / ]p", "previous / next paragraph"),
        ],
    ),
    (
        "02  EDITING & CHANGES",
        [
            ("i / a", "insert before / append after"),
            ("I / A", "insert at line start / end"),
            ("o / O", "open line below / above"),
            ("d / c", "delete / change selection"),
            ("y / p / P", "yank / paste after / before"),
            ("R", "replace with yanked text"),
            ("r<char>", "replace with character"),
            ("u / U", "undo / redo"),
            ("> / <", "indent / unindent"),
            ("J", "join selected lines"),
            ("Ctrl+c", "toggle comment"),
            ("=", "format selection"),
            ('"<reg>', "choose register for yank/paste"),
            ("Space+y / Space+p", "clipboard yank / paste"),
        ],
    ),
    (
        "03  SELECTION & SEARCH",
        [
            ("v", "extend selection; Esc exits"),
            ("x / X", "select line / extend to bounds"),
            ("; / Alt+;", "collapse / flip selection"),
            ("%", "select whole document"),
            ("s / S", "select matches / split on matches"),
            ("K / Alt+K", "keep / remove matching selections"),
            (", / Alt+,", "keep / remove primary selection"),
            ("C / Alt+C", "copy selection below / above"),
            ("/ / ?", "search forward / backward (regex)"),
            ("n / N", "next / previous search match"),
            ("* / Alt+*", "search selection bounded / exact"),
            ("mm", "go to matching bracket"),
            ("ma<char> / mi<char>", "select around / inside object"),
            ("Alt+o / Alt+i", "expand / shrink syntax selection"),
        ],
    ),
    (
        "04  SPECIAL MODES & WINDOWS",
        [
            ("Ctrl+w s / v", "horizontal / vertical split"),
            ("Ctrl+w h j k l", "move between splits"),
            ("Ctrl+w q / o", "close pane / close other panes"),
            ("Ctrl+w w / z", "next pane / toggle maximize"),
            ("Ctrl+w r", "resize mode; arrows resize"),
            ("Space+f / b", "file / buffer picker"),
            ("Space+/", "search the workspace"),
            ("Space+e", "workspace file explorer"),
            ("Picker: Tab / Shift+Tab", "next / previous item"),
            ("Picker: Ctrl+s / Ctrl+v", "open in horizontal / vertical split"),
            ("Insert: Ctrl+h / Ctrl+d", "delete previous / next char"),
            ("Insert: Ctrl+w", "delete previous word"),
            ("Terminal: Ctrl+\\", "leader (Space goes to shell)"),
            ("Image: + / - / 0", "zoom in / out / fit"),
        ],
    ),
]

INK = HexColor("#17212B")
PAPER = HexColor("#F5F1E8")
CARD = HexColor("#FFFDF7")
MUTED = HexColor("#68737D")
CYAN = HexColor("#1B8793")
CORAL = HexColor("#E85D4A")
LINE = HexColor("#D9D4C8")


def draw_key(c, x, y, label, size=7.2):
    """Draw a compact keycap and return its width."""
    width = stringWidth(label, "Courier-Bold", size) + 10
    c.setFillColor(PAPER)
    c.roundRect(x, y - 3, width, 14, 3, fill=1, stroke=0)
    c.setFillColor(INK)
    c.setFont("Courier-Bold", size)
    c.drawCentredString(x + width / 2, y + 1, label)
    return width


def draw_essentials(c, x, y, width):
    c.setFillColor(INK)
    c.roundRect(x, y, width, 57, 7, fill=1, stroke=0)
    c.setFillColor(CARD)
    c.setFont("Helvetica-Bold", 8)
    c.drawString(x + 14, y + 40, "ABSOLUTE ESSENTIALS")

    item_width = (width - 28) / len(ESSENTIALS)
    for i, (key, action) in enumerate(ESSENTIALS):
        item_x = x + 14 + i * item_width
        key_width = draw_key(c, item_x, y + 17, key)
        c.setFillColor(CARD)
        c.setFont("Helvetica", 7.3)
        c.drawString(item_x + key_width + 5, y + 18, action)


def draw_section(c, x, y, width, height, title, rows, accent):
    c.setFillColor(CARD)
    c.roundRect(x, y, width, height, 7, fill=1, stroke=0)
    c.setFillColor(accent)
    c.roundRect(x, y + height - 27, width, 27, 7, fill=1, stroke=0)
    c.rect(x, y + height - 27, width, 7, fill=1, stroke=0)
    c.setFillColor(CARD)
    c.setFont("Helvetica-Bold", 8.4)
    c.drawString(x + 13, y + height - 18, title)

    split = (len(rows) + 1) // 2
    columns = (rows[:split], rows[split:])
    inner_width = (width - 34) / 2
    row_height = (height - 41) / split
    for column, entries in enumerate(columns):
        column_x = x + 13 + column * (inner_width + 8)
        if column:
            c.setStrokeColor(LINE)
            c.setLineWidth(0.5)
            c.line(x + width / 2, y + 10, x + width / 2, y + height - 37)
        for row, (key, action) in enumerate(entries):
            row_y = y + height - 43 - row * row_height
            key_width = draw_key(c, column_x, row_y, key)
            c.setFillColor(INK)
            c.setFont("Helvetica", 7.2)
            c.drawString(column_x + key_width + 5, row_y + 1, action)


def generate():
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    page_width, page_height = landscape(A4)
    c = canvas.Canvas(str(OUTPUT), pagesize=(page_width, page_height))
    c.setTitle("Toe Cheatsheet")
    c.setAuthor("Toe contributors")
    c.setSubject("Thom's Own Editor keyboard reference")

    c.setFillColor(PAPER)
    c.rect(0, 0, page_width, page_height, fill=1, stroke=0)

    margin = 24
    c.setFillColor(INK)
    c.setFont("Helvetica-Bold", 25)
    c.drawString(margin, page_height - 40, "TOE")
    c.setFillColor(CORAL)
    c.circle(margin + 66, page_height - 33, 3, fill=1, stroke=0)
    c.setFillColor(INK)
    c.setFont("Helvetica-Bold", 10)
    c.drawString(margin + 78, page_height - 36, "QUICK REFERENCE")
    c.setFillColor(MUTED)
    c.setFont("Helvetica", 7.5)
    c.drawRightString(
        page_width - margin,
        page_height - 36,
        "Normal-mode defaults  |  counts repeat commands  |  "
        "Space opens leader",
    )

    draw_essentials(c, margin, page_height - 111, page_width - margin * 2)

    gap = 10
    card_width = (page_width - margin * 2 - gap) / 2
    card_height = 198
    top_y = page_height - 321
    bottom_y = 25
    positions = [
        (margin, top_y),
        (margin + card_width + gap, top_y),
        (margin, bottom_y),
        (margin + card_width + gap, bottom_y),
    ]
    for i, ((title, rows), (x, y)) in enumerate(zip(SECTIONS, positions)):
        draw_section(
            c,
            x,
            y,
            card_width,
            card_height,
            title,
            rows,
            CYAN if i % 2 == 0 else CORAL,
        )

    c.setFillColor(MUTED)
    c.setFont("Helvetica", 6.7)
    c.drawRightString(
        page_width - margin,
        12,
        "Full reference: toe docs > Key Bindings  |  Ctrl+\\ is also a leader",
    )
    c.save()


if __name__ == "__main__":
    generate()
