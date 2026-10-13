"""Render local IDEF0-SVG models before MkDocs copies static assets."""

from __future__ import annotations

import json
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
MODELS = ROOT / "docs" / "idef0-svg" / "models"
RENDERER = ROOT / "docs" / "tools" / "idef0-svg" / "render.rb"
VIEW_NAME = re.compile(r"[a-z0-9]+(?:-[a-z0-9]+)*\Z")


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


def render_models(check: bool = False) -> int:
    count = 0
    for source in sorted(MODELS.rglob("*.idef0")):
        style = source.with_suffix(".idef0.json")
        options = json.loads(style.read_text(encoding="utf-8")) if style.exists() else {}
        if not isinstance(options, dict):
            raise ValueError(f"{style}: expected an object")
        render_view(source, style, options.get("mode", "decompose"),
                    options.get("process"), source.with_suffix(".svg"), check)
        count += 1
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


def on_pre_build(config):
    render_models()


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="check generated SVG without writing")
    args = parser.parse_args()
    print(f"IDEF0-SVG: {render_models(check=args.check)} diagram(s)")
