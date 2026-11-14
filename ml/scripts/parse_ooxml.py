#!/usr/bin/env python3
"""parse_ooxml.py — извлечение текста из DOCX и XLSX в унифицированный .txt.

DOCX: абзацы word/document.xml, таблицы превращаются в строки «клетка | клетка».
XLSX: «Лист: <имя>» + строки «ячейка=значение» (как Go-парсер backend'а),
плюс сохранение объединённых ячеек как повторов значения — упрощает поиск.

Только stdlib. Использование: parse_ooxml.py <dir_src> <dir_out>
"""
import re
import sys
import zipfile
import xml.etree.ElementTree as ET
from pathlib import Path

W = "{http://schemas.openxmlformats.org/wordprocessingml/2006/main}"
NS = "{http://schemas.openxmlformats.org/spreadsheetml/2006/main}"


def docx_text(path: Path) -> str:
    with zipfile.ZipFile(path) as z:
        xml_data = z.read("word/document.xml")
    root = ET.fromstring(xml_data)
    body = root.find(f"{W}body")
    out = []

    def para_text(p):
        parts = []
        for node in p.iter():
            if node.tag == f"{W}t" and node.text:
                parts.append(node.text)
            elif node.tag == f"{W}tab":
                parts.append("\t")
            elif node.tag == f"{W}br":
                parts.append("\n")
        return "".join(parts).strip()

    for el in body:
        if el.tag == f"{W}p":
            t = para_text(el)
            out.append(t)
        elif el.tag == f"{W}tbl":
            for row in el.iter(f"{W}tr"):
                cells = [para_text(tc) or "" for tc in row.iter(f"{W}tc")]
                out.append(" | ".join(cells).strip(" |"))
    return "\n".join(out).strip()


def col_to_num(col: str) -> int:
    n = 0
    for ch in col:
        n = n * 26 + (ord(ch) - 64)
    return n


def xlsx_text(path: Path) -> str:
    with zipfile.ZipFile(path) as z:
        shared = []
        if "xl/sharedStrings.xml" in z.namelist():
            ss = ET.fromstring(z.read("xl/sharedStrings.xml"))
            for si in ss.iter(f"{NS}si"):
                shared.append("".join(t.text or "" for t in si.iter(f"{NS}t")))
        wb = ET.fromstring(z.read("xl/workbook.xml"))
        names = [s.get("name", f"sheet{i}") for i, s in enumerate(wb.iter(f"{NS}sheet"))]

        out = []
        sheets = sorted(
            (n for n in z.namelist() if re.fullmatch(r"xl/worksheets/sheet\d+\.xml", n)),
            key=lambda n: int(re.search(r"\d+", n).group()),
        )
        for i, sname in enumerate(sheets):
            title = names[i] if i < len(names) else sname
            out.append(f"Лист: {title}")
            root = ET.fromstring(z.read(sname))
            for row in root.iter(f"{NS}row"):
                cells = []
                for c in row.iter(f"{NS}c"):
                    ref = c.get("r", "")
                    t = c.get("t")
                    v = c.find(f"{NS}v")
                    is_el = c.find(f"{NS}is")
                    val = ""
                    if t == "s" and v is not None and v.text is not None:
                        idx = int(v.text)
                        val = shared[idx] if idx < len(shared) else ""
                    elif t == "inlineStr" and is_el is not None:
                        val = "".join(x.text or "" for x in is_el.iter(f"{NS}t"))
                    elif v is not None and v.text is not None:
                        val = v.text
                    if val != "":
                        cells.append(f"{ref}={val}")
                if cells:
                    out.append("\t".join(cells))
            out.append("")
        return "\n".join(out).strip()


def main() -> int:
    src, dst = Path(sys.argv[1]), Path(sys.argv[2])
    dst.mkdir(parents=True, exist_ok=True)
    for f in sorted(src.iterdir()):
        ext = f.suffix.lower()
        try:
            if ext == ".docx":
                text = docx_text(f)
            elif ext == ".xlsx":
                text = xlsx_text(f)
            else:
                continue
            (dst / (f.stem + ".txt")).write_text(text + "\n", encoding="utf-8")
            print(f"{len(text):>8}  {f.name}")
        except Exception as e:
            print(f"FAIL {f.name}: {e}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
