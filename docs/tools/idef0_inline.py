"""Render IDEF0 code fences directly inside Markdown pages."""

from __future__ import annotations

import html
import json
import logging
import re
import subprocess
import tempfile
from pathlib import Path
from typing import Callable

from idef0_render import RENDERER, ROOT

FENCE = re.compile(r"^ {0,3}(?P<mark>`{3,}|~{3,})(?P<info>[^`~]*)$")
VIEW_MODES = {"auto", "focus", "decompose", "schematic"}
KIND = {"input", "control", "output", "mechanism"}
STYLE_KEYS = {
    "@defaults": {"arrow", "arrow_text", "block_fill", "block_stroke", "block_text"},
    "@block": {"fill", "stroke", "text"},
    "@arrow": {"stroke", "text"},
}
EXPORT_NAME = re.compile(r"[a-z0-9]+(?:-[a-z0-9]+)*\.svg\Z")
LOG = logging.getLogger("mkdocs.idef0")
_serve_mode = False


def _message(error: Exception, page_path: str, line: int) -> str:
    message = str(error).strip() or type(error).__name__
    if message.startswith(f"{page_path}:"):
        return message
    return str(_error(page_path, line, message))


def _error_panel(message: str) -> str:
    safe = html.escape(message)
    return (f'<aside class="idef0-error" role="alert"><strong>Ошибка IDEF0</strong>'
            f'<pre>{safe}</pre><p>Исправьте исходник диаграммы и сохраните страницу.</p></aside>')


def _insert_banner(markdown: str, errors: list[str]) -> str:
    if not errors:
        return markdown
    items = "".join(f"<li><code>{html.escape(message)}</code></li>" for message in errors)
    banner = (f'<aside class="idef0-error" role="alert"><strong>Ошибки генерации IDEF0</strong>'
              f'<ul>{items}</ul><p>Исправьте исходные файлы и сохраните их.</p></aside>')
    heading = re.search(r"(?m)^# [^\n]+\n", markdown)
    if heading:
        return markdown[:heading.end()] + "\n" + banner + "\n\n" + markdown[heading.end():]
    return banner + "\n\n" + markdown


def _error(page_path: str, line: int, message: str) -> ValueError:
    return ValueError(f"{page_path}:{line}: IDEF0: {message}")


def _style_values(raw: str, allowed: set[str], page_path: str, line: int) -> dict[str, str]:
    values = {}
    for token in raw.split():
        key, separator, value = token.partition("=")
        if not separator or key not in allowed or not value or key in values:
            raise _error(page_path, line, f"invalid color setting {token!r}")
        values[key] = value
    if not values:
        raise _error(page_path, line, "at least one color setting is required")
    return values


def _parse_block(body: str, page_path: str, first_line: int):
    views = []
    settings: dict = {}
    statements = []
    statement_lines = []
    export = None
    for offset, raw in enumerate(body.splitlines(), start=1):
        line_number = first_line + offset
        line = raw.strip()
        if not line:
            continue
        if line.startswith("@view"):
            fields = line.split(maxsplit=2)
            if len(fields) < 2 or fields[1] not in VIEW_MODES:
                raise _error(page_path, line_number,
                             "use @view auto|focus|decompose|schematic [Process Name]")
            views.append((fields[1], fields[2] if len(fields) > 2 else None))
        elif line.startswith("@export"):
            name = line[len("@export"):].strip()
            if export is not None or not EXPORT_NAME.fullmatch(name):
                raise _error(page_path, line_number, "use one @export file-name.svg per block")
            export = name
        elif line.startswith("@tunnel"):
            fields = [field.strip() for field in line[len("@tunnel"):].split("|")]
            if len(fields) != 4 or fields[0] not in {"parent", "child"} or fields[2] not in KIND \
                    or not fields[1] or not fields[3]:
                raise _error(page_path, line_number,
                             "use @tunnel parent|child | Process Name | input|control|output|mechanism | Arrow Name")
            side, process, kind, name = fields
            entry = settings.setdefault("boundaries", {}).setdefault(process, {})
            names = entry.setdefault(f"{side}_tunnels", {}).setdefault(kind, [])
            if name in names:
                raise _error(page_path, line_number, f"duplicate tunnel for {process}: {kind} {name}")
            names.append(name)
        elif line.startswith("@defaults"):
            values = _style_values(line[len("@defaults"):].strip(), STYLE_KEYS["@defaults"],
                                   page_path, line_number)
            settings.setdefault("defaults", {}).update(values)
        elif line.startswith("@block ") or line.startswith("@arrow "):
            directive, rest = line.split(maxsplit=1)
            name, separator, raw_values = rest.partition("|")
            if not separator or not name.strip():
                raise _error(page_path, line_number,
                             f"use {directive} Name | color=value [color=value]")
            values = _style_values(raw_values.strip(), STYLE_KEYS[directive], page_path, line_number)
            group = "blocks" if directive == "@block" else "arrows"
            settings.setdefault(group, {}).setdefault(name.strip(), {}).update(values)
        elif line.startswith("@"):
            raise _error(page_path, line_number, f"unknown directive {line.split()[0]}")
        else:
            statements.append(raw)
            statement_lines.append(line_number)
    if not statements:
        raise _error(page_path, first_line, "diagram has no IDEF0 statements")
    return "\n".join(statements) + "\n", settings, views or [("auto", None)], export, statement_lines


def _render_block(body: str, page_path: str, first_line: int,
                  export_dir: Path | None = None, check: bool = False,
                  registry: set[Path] | None = None) -> str:
    model, settings, views, export, statement_lines = _parse_block(body, page_path, first_line)
    if export_dir is not None and export is None:
        return ""
    figures = []
    with tempfile.TemporaryDirectory(prefix="careeria-idef0-") as directory:
        source = Path(directory) / "model.idef0"
        source.write_text(model, encoding="utf-8")
        style = Path(directory) / "model.idef0.json"
        if settings:
            style.write_text(json.dumps(settings, ensure_ascii=False), encoding="utf-8")
        for index, (mode, process) in enumerate(views, start=1):
            command = ["ruby", str(RENDERER), "--input", str(source), "--mode", mode]
            if settings:
                command += ["--style", str(style)]
            if process:
                command += ["--process", process]
            result = subprocess.run(command, capture_output=True, text=True, check=False)
            if result.returncode:
                message = result.stderr.strip() or "render failed"
                model_line = re.search(r"\bline (\d+):", message)
                line = first_line
                if model_line and 1 <= int(model_line.group(1)) <= len(statement_lines):
                    line = statement_lines[int(model_line.group(1)) - 1]
                raise _error(page_path, line, message)
            if export_dir is not None and export is not None:
                suffix = "" if index == 1 else f"-{index}"
                target = export_dir / f"{export[:-4]}{suffix}.svg"
                if registry is not None:
                    if target in registry:
                        raise _error(page_path, first_line, f"duplicate SVG export: {target.name}")
                    registry.add(target)
                if not target.exists() or target.read_text(encoding="utf-8") != result.stdout:
                    if check:
                        raise _error(page_path, first_line,
                                     f"{target} is stale or missing; run make idef0-render")
                    target.write_text(result.stdout, encoding="utf-8")
            start = result.stdout.find("<svg ")
            if start < 0:
                raise _error(page_path, first_line, "renderer did not return SVG")
            svg = result.stdout[start:].strip()
            title = {"focus": "Контекст", "decompose": "Декомпозиция",
                     "schematic": "Обзор"}.get(mode, "Диаграмма")
            label = f"{title}: {process}" if process else "Диаграмма IDEF0"
            svg = svg.replace("<svg ",
                              f'<svg role="img" aria-label="{html.escape(label, quote=True)}" ', 1)
            caption = ""
            if len(views) > 1:
                caption = f"<figcaption>{html.escape(label)}</figcaption>"
            figures.append(f'<figure class="idef0-inline">{svg}{caption}</figure>')
    return "\n".join(figures)


def render_inline_diagrams(markdown: str, page_path: str, show_errors: bool = False) -> str:
    lines = markdown.splitlines(keepends=True)
    output = []
    index = 0
    while index < len(lines):
        opening = FENCE.fullmatch(lines[index].rstrip("\r\n"))
        if not opening:
            output.append(lines[index])
            index += 1
            continue
        mark = opening.group("mark")
        is_idef0 = opening.group("info").strip() == "idef0"
        closing = re.compile(rf"^ {{0,3}}{re.escape(mark[0])}{{{len(mark)},}}\s*$")
        end = index + 1
        while end < len(lines) and not closing.fullmatch(lines[end].rstrip("\r\n")):
            end += 1
        if end == len(lines):
            if is_idef0:
                error = _error(page_path, index + 1, "unclosed idef0 code fence")
                if not show_errors:
                    raise error
                LOG.error("%s", error)
                output.append("\n" + _error_panel(str(error)) + "\n")
                break
            output.extend(lines[index:])
            break
        if is_idef0:
            body = "".join(lines[index + 1:end])
            try:
                rendered = _render_block(body, page_path, index + 1)
            except Exception as error:
                if not show_errors:
                    raise
                message = _message(error, page_path, index + 1)
                LOG.error("%s", message)
                rendered = _error_panel(message)
            output.append("\n" + rendered + "\n")
        else:
            output.extend(lines[index:end + 1])
        index = end + 1
    return "".join(output)


def export_markdown_diagrams(docs_dir: Path, check: bool = False,
                             on_error: Callable[[Exception], None] | None = None) -> int:
    count = 0
    registry: set[Path] = set()
    for page in sorted(docs_dir.rglob("*.md")):
        markdown = page.read_text(encoding="utf-8")
        lines = markdown.splitlines(keepends=True)
        index = 0
        while index < len(lines):
            opening = FENCE.fullmatch(lines[index].rstrip("\r\n"))
            if not opening:
                index += 1
                continue
            mark = opening.group("mark")
            closing = re.compile(rf"^ {{0,3}}{re.escape(mark[0])}{{{len(mark)},}}\s*$")
            end = index + 1
            while end < len(lines) and not closing.fullmatch(lines[end].rstrip("\r\n")):
                end += 1
            if end == len(lines):
                if opening.group("info").strip() == "idef0":
                    error = _error(str(page.relative_to(docs_dir)), index + 1,
                                   "unclosed idef0 code fence")
                    if on_error is None:
                        raise error
                    on_error(error)
                break
            if opening.group("info").strip() == "idef0":
                body = "".join(lines[index + 1:end])
                if any(line.strip().startswith("@export") for line in body.splitlines()):
                    try:
                        _render_block(body, str(page.relative_to(docs_dir)), index + 1,
                                      page.parent, check, registry)
                        count += 1
                    except Exception as error:
                        if on_error is None:
                            raise
                        on_error(ValueError(_message(error, str(page.relative_to(docs_dir)), index + 1)))
            index = end + 1
    return count


def on_startup(*, command, dirty):
    global _serve_mode
    _serve_mode = command == "serve"


def on_pre_build(config):
    if not _serve_mode:
        export_markdown_diagrams(Path(config.docs_dir))
        return
    errors = []

    def record(error: Exception) -> None:
        errors.append(str(error))
        LOG.error("%s", error)

    export_markdown_diagrams(Path(config.docs_dir), on_error=record)
    config.extra.setdefault("_idef0_errors", {})["exports"] = errors


def on_page_markdown(markdown, *, page, config, files):
    result = render_inline_diagrams(markdown, page.file.src_uri, show_errors=_serve_mode)
    if _serve_mode:
        diagnostics = config.extra.get("_idef0_errors", {})
        global_errors = diagnostics.get("models", []) + diagnostics.get("exports", [])
        result = _insert_banner(result, global_errors)
    return result


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="check exported SVGs without writing")
    args = parser.parse_args()
    print(f"IDEF0 inline: {export_markdown_diagrams(ROOT / 'docs', args.check)} exported block(s)")
