"""Render local IDEF0-SVG models before MkDocs copies static assets."""

from __future__ import annotations

import json
import html
import logging
import re
import subprocess
import sys
import textwrap
from pathlib import Path
from typing import Callable

ROOT = Path(__file__).resolve().parents[2]
MODELS = ROOT / "docs" / "idef0-svg" / "models"
RENDERER = ROOT / "docs" / "tools" / "idef0-svg" / "render.rb"
VIEW_NAME = re.compile(r"[a-z0-9]+(?:-[a-z0-9]+)*\Z")
LOG = logging.getLogger("mkdocs.idef0")


def write_error_svg(target: Path, message: str) -> None:
    lines = textwrap.wrap(message.replace("\n", " "), width=100, break_long_words=True) or ["Ошибка IDEF0"]
    lines = lines[:12]
    height = max(120, 75 + len(lines) * 23)
    labels = "\n".join(
        f'<text x="24" y="{76 + index * 23}" font-size="15" fill="#7f1d1d">'
        f'{html.escape(line)}</text>' for index, line in enumerate(lines)
    )
    svg = (f'<svg xmlns="http://www.w3.org/2000/svg" width="900" height="{height}" '
           f'viewBox="0 0 900 {height}" role="img" aria-label="Ошибка IDEF0">'
           f'<rect x="1" y="1" width="898" height="{height - 2}" rx="8" '
           f'fill="#fff1f0" stroke="#c62828" stroke-width="2"/>'
           f'<text x="24" y="38" font-size="22" font-weight="bold" fill="#7f1d1d">Ошибка IDEF0</text>'
           f'{labels}</svg>\n')
    if not target.exists() or target.read_text(encoding="utf-8") != svg:
        target.write_text(svg, encoding="utf-8")


def _error_targets(source: Path) -> set[Path]:
    targets = {source.with_suffix(".svg")}
    style = source.with_suffix(".idef0.json")
    if style.exists():
        try:
            options = json.loads(style.read_text(encoding="utf-8"))
            for view in options.get("views", []):
                name = view.get("name")
                if isinstance(name, str) and VIEW_NAME.fullmatch(name):
                    targets.add(source.with_name(f"{source.stem}-{name}.svg"))
        except (ValueError, AttributeError, TypeError):
            pass
    targets.update(source.parent.glob(f"{source.stem}-*.svg"))
    return targets


def render_view(source: Path, style: Path, mode: str, process: str | None, target: Path, check: bool) -> None:
    if mode not in ("decompose", "schematic", "focus"):
        raise ValueError(f"{style}: invalid mode {mode!r}")
    if process is not None and (not isinstance(process, str) or not process.strip()):
        raise ValueError(f"{style}: process must be non-empty text")
    command = ["ruby", str(RENDERER), "--input", str(source), "--mode", mode]
    if style.exists():
        command += ["--style", str(style)]
    if process:
        command += ["--process", process]
    result = subprocess.run(command, capture_output=True, text=True, check=False)
    if result.returncode:
        raise RuntimeError(f"{source}: IDEF0-SVG failed:\n{result.stderr.strip()}")
    if not target.exists() or target.read_text(encoding="utf-8") != result.stdout:
        if check:
            raise RuntimeError(f"{target}: stale or missing; run make idef0-render")
        target.write_text(result.stdout, encoding="utf-8")


def _render_model(source: Path, check: bool) -> int:
    style = source.with_suffix(".idef0.json")
    try:
        options = json.loads(style.read_text(encoding="utf-8")) if style.exists() else {}
    except json.JSONDecodeError as error:
        raise ValueError(f"{style}:{error.lineno}:{error.colno}: invalid JSON: {error.msg}") from error
    if not isinstance(options, dict):
        raise ValueError(f"{style}: expected an object")
    render_view(source, style, options.get("mode", "decompose"),
                options.get("process"), source.with_suffix(".svg"), check)
    count = 1
    views = options.get("views", [])
    if not isinstance(views, list):
        raise ValueError(f"{style}: views must be a list")
    names = set()
    for view in views:
        if not isinstance(view, dict) or set(view) - {"name", "mode", "process"}:
            raise ValueError(f"{style}: each view needs name, mode and process fields only")
        name = view.get("name")
        if not isinstance(name, str) or not VIEW_NAME.fullmatch(name) or name in names:
            raise ValueError(f"{style}: view name must be unique lowercase words separated by hyphens")
        names.add(name)
        target = source.with_name(f"{source.stem}-{name}.svg")
        render_view(source, style, view.get("mode", "decompose"),
                    view.get("process"), target, check)
        count += 1
    return count


def render_models(check: bool = False,
                  on_error: Callable[[Path, Exception], None] | None = None) -> int:
    count = 0
    for source in sorted(MODELS.rglob("*.idef0")):
        try:
            count += _render_model(source, check)
        except Exception as error:
            if on_error is None:
                raise
            on_error(source, error)
    return count


def on_pre_build(config):
    errors = []

    def record(source: Path, error: Exception) -> None:
        message = str(error).replace(f"{ROOT}/", "")
        try:
            relative_source = str(source.relative_to(ROOT))
        except ValueError:
            relative_source = str(source)
        if not message.startswith(relative_source) and not message.startswith("docs/"):
            message = f"{relative_source}: {message}"
        try:
            source_uri = source.relative_to(ROOT / "docs").as_posix()
        except ValueError:
            source_uri = str(source)
        errors.append({"source": source_uri, "message": message})
        LOG.info("Ошибка IDEF0: %s", message)
        for target in _error_targets(source):
            write_error_svg(target, message)

    render_models(on_error=record)
    config.extra.setdefault("_idef0_errors", {})["models"] = errors


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="check generated SVG without writing")
    args = parser.parse_args()
    def report(source: Path, error: Exception) -> None:
        print(f"IDEF0: {source}: {error}", file=sys.stderr)
        if not args.check:
            for target in _error_targets(source):
                write_error_svg(target, f"{source}: {error}")

    print(f"IDEF0-SVG: {render_models(check=args.check, on_error=report)} diagram(s)")
