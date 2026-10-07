#!/usr/bin/env python3
"""Invalidate Markdown pages when a transitive PlantUML include changes.

MkDocs tracks files in ``docs/``, but plantuml-markdown does not expose its
``!include`` graph to MkDocs. With ``--dirtyreload`` an edit in common/*.puml
would otherwise leave pages embedding a diagram with a stale SVG. This watcher
touches only the affected Markdown source pages, so MkDocs rebuilds those pages
without a full site rebuild.
"""

from __future__ import annotations

import os
import re
import time
from pathlib import Path


DOCS = Path(__file__).resolve().parents[1]
INCLUDE = re.compile(r"^\s*!include(?:sub)?\s+([^\s]+)", re.MULTILINE)


def local_includes(source: Path, root_relative: bool = False) -> set[Path]:
    """Return local PlantUML files included by a source file."""
    result: set[Path] = set()
    try:
        content = source.read_text(encoding="utf-8")
    except FileNotFoundError:
        return result

    for match in INCLUDE.finditer(content):
        include = match.group(1).strip('"').split("!", 1)[0]
        if include.startswith("<"):
            continue
        candidate = (DOCS if root_relative else source.parent) / include
        candidate = candidate.resolve()
        if candidate.suffix == ".puml" and candidate.is_file():
            result.add(candidate)
    return result


def pages_affected_by(changed: set[Path]) -> set[Path]:
    """Find Markdown pages whose embedded diagram includes a changed file."""
    affected: set[Path] = set()
    memo: dict[Path, set[Path]] = {}

    def dependencies(diagram: Path, seen: set[Path] | None = None) -> set[Path]:
        if diagram in memo:
            return memo[diagram]
        seen = seen or set()
        if diagram in seen:
            return set()
        direct = local_includes(diagram)
        resolved = set(direct)
        for dependency in direct:
            resolved.update(dependencies(dependency, seen | {diagram}))
        memo[diagram] = resolved
        return resolved

    for page in DOCS.rglob("*.md"):
        diagrams = local_includes(page, root_relative=True)
        if any(changed & (dependencies(diagram) | {diagram}) for diagram in diagrams):
            affected.add(page)
    return affected


def puml_snapshot() -> dict[Path, int]:
    return {
        path.resolve(): path.stat().st_mtime_ns
        for path in DOCS.rglob("*.puml")
        if path.is_file()
    }


def main() -> None:
    previous = puml_snapshot()
    while True:
        time.sleep(0.4)
        current = puml_snapshot()
        changed = {
            path
            for path in previous.keys() | current.keys()
            if previous.get(path) != current.get(path)
        }
        if changed:
            for page in pages_affected_by(changed):
                os.utime(page, None)
            previous = current


if __name__ == "__main__":
    main()
