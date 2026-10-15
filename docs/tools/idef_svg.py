#!/usr/bin/env python3
"""Generate IDEF0 SVG diagrams from *.idef.json models, with no third-party packages."""

from __future__ import annotations

import argparse
import json
import re
import sys
import textwrap
from html import escape
from pathlib import Path

KINDS = {"input", "control", "output", "mechanism", "flow"}
INCOMING = {"input", "control", "mechanism"}
COLOR = re.compile(r"^#[0-9a-fA-F]{3}(?:[0-9a-fA-F]{3})?$|^[a-zA-Z]+$")
ID = re.compile(r"^A(?:-0|[0-9]+)$")
ARROW_ID = re.compile(r"^[A-Za-z][A-Za-z0-9_-]*$")
DEFAULT_THEME = {
    "background": "#ffffff",
    "frame": "#b8c6d3",
    "title": "#172b41",
    "subtitle": "#45576b",
    "block_fill": "#f8fbfe",
    "block_stroke": "#26384b",
    "block_text": "#172b41",
    "block_number": "#26384b",
    "arrow": "#26384b",
    "arrow_text": "#172b41",
}
BLOCK_STYLE = {"fill", "stroke", "text", "number"}
ARROW_STYLE = {"stroke", "text"}


class ModelError(ValueError):
    pass


def fail(where: str, message: str) -> None:
    raise ModelError(f"{where}: {message}")


def require_object(value, where):
    if not isinstance(value, dict):
        fail(where, "expected an object")
    return value


def require_text(value, where):
    if not isinstance(value, str) or not value.strip():
        fail(where, "expected non-empty text")
    return value.strip()


def check_colors(values, allowed, where):
    require_object(values, where)
    for key, color in values.items():
        if key not in allowed:
            fail(where, f"unknown color field {key!r}")
        if not isinstance(color, str) or not COLOR.fullmatch(color):
            fail(where, f"invalid color {key}={color!r}; use #RGB, #RRGGBB or a CSS color name")


def incident_roles(diagram, block_id):
    roles = set()
    for arrow in diagram["arrows"]:
        if arrow.get("tunnel"):
            continue
        kind = arrow["kind"]
        if kind == "flow":
            if arrow["from"] == block_id:
                roles.add((arrow["id"], "output"))
            if arrow["to"] == block_id:
                roles.add((arrow["id"], arrow.get("to_kind", "input")))
        elif kind == "output" and arrow["from"] == block_id:
            roles.add((arrow["id"], "output"))
        elif kind in INCOMING and arrow["to"] == block_id:
            roles.add((arrow["id"], kind))
    return roles


def validate(model):
    require_object(model, "model")
    require_text(model.get("title"), "model.title")
    theme = model.get("theme", {})
    check_colors(theme, set(DEFAULT_THEME), "model.theme")
    diagrams = model.get("diagrams")
    if not isinstance(diagrams, list) or not diagrams:
        fail("model.diagrams", "expected a non-empty list")
    by_id = {}
    for diagram in diagrams:
        require_object(diagram, "diagram")
        did = require_text(diagram.get("id"), "diagram.id")
        if not ID.fullmatch(did):
            fail(did, "diagram id must be A-0, A0, A1, A11, ...")
        if did in by_id:
            fail(did, "duplicate diagram id")
        by_id[did] = diagram
        require_text(diagram.get("title"), f"{did}.title")
        check_colors(diagram.get("theme", {}), set(DEFAULT_THEME), f"{did}.theme")
        blocks = diagram.get("blocks")
        if not isinstance(blocks, list) or not blocks:
            fail(did, "expected a non-empty blocks list")
        if did == "A-0" and (len(blocks) != 1 or blocks[0].get("id") != "A0"):
            fail(did, "context diagram must contain exactly block A0")
        if did != "A-0" and not 3 <= len(blocks) <= 6:
            fail(did, "IDEF0 decomposition must contain 3–6 blocks")
        block_ids = set()
        for number, block in enumerate(blocks, 1):
            require_object(block, f"{did}.block")
            bid = require_text(block.get("id"), f"{did}.block.id")
            expected = "A0" if did == "A-0" else f"{did}{number}"
            if bid != expected:
                fail(did, f"block {number} must have id {expected}, got {bid}")
            block_ids.add(bid)
            require_text(block.get("label"), f"{did}.{bid}.label")
            check_colors(block.get("style", {}), BLOCK_STYLE, f"{did}.{bid}.style")
            if "child" in block and block["child"] != bid:
                fail(did, f"{bid}.child must be {bid}")
        arrows = diagram.get("arrows", [])
        if not isinstance(arrows, list):
            fail(did, "arrows must be a list")
        arrow_ids = set()
        for arrow in arrows:
            require_object(arrow, f"{did}.arrow")
            aid = require_text(arrow.get("id"), f"{did}.arrow.id")
            if not ARROW_ID.fullmatch(aid) or aid in arrow_ids:
                fail(did, f"invalid or duplicate arrow id {aid!r}")
            arrow_ids.add(aid)
            kind = arrow.get("kind")
            if kind not in KINDS:
                fail(did, f"{aid}: kind must be one of {', '.join(sorted(KINDS))}")
            require_text(arrow.get("label"), f"{did}.{aid}.label")
            check_colors(arrow.get("style", {}), ARROW_STYLE, f"{did}.{aid}.style")
            if "tunnel" in arrow and not isinstance(arrow["tunnel"], bool):
                fail(did, f"{aid}.tunnel must be true or false")
            if kind in INCOMING:
                if arrow.get("to") not in block_ids or "from" in arrow:
                    fail(did, f"{aid}: {kind} must have one valid 'to' block")
            elif kind == "output":
                if arrow.get("from") not in block_ids or "to" in arrow:
                    fail(did, f"{aid}: output must have one valid 'from' block")
            else:
                if arrow.get("from") not in block_ids or arrow.get("to") not in block_ids:
                    fail(did, f"{aid}: flow requires valid 'from' and 'to' blocks")
                if arrow["from"] == arrow["to"]:
                    fail(did, f"{aid}: self flow is unsupported")
                if arrow.get("to_kind", "input") not in INCOMING:
                    fail(did, f"{aid}.to_kind must be input, control or mechanism")
    if "A-0" not in by_id:
        fail("model", "missing context diagram A-0")
    visited = set()
    def walk(did):
        if did in visited:
            fail(did, "diagram is referenced more than once or hierarchy contains a cycle")
        visited.add(did)
        diagram = by_id[did]
        for block in diagram["blocks"]:
            child = block.get("child")
            if child:
                if child not in by_id:
                    fail(did, f"missing child diagram {child}")
                expected = incident_roles(diagram, block["id"])
                actual = {(a["id"], a["kind"]) for a in by_id[child]["arrows"]
                          if a["kind"] != "flow" and not a.get("tunnel")}
                if expected != actual:
                    missing = sorted(expected - actual)
                    extra = sorted(actual - expected)
                    fail(child, f"boundary arrows do not balance with {did}.{block['id']}; missing={missing}, extra={extra}")
                walk(child)
    walk("A-0")
    if set(by_id) != visited:
        fail("model", f"unreachable diagrams: {', '.join(sorted(set(by_id) - visited))}")
    return by_id


def lines(value, width=23):
    result = []
    for paragraph in value.split("\n"):
        result.extend(textwrap.wrap(paragraph, width=width, break_long_words=False) or [""])
    return result


def text(x, y, label, color, size=16, anchor="middle", weight="500", width=23, leading=None):
    result = []
    leading = leading or size * 1.25
    for index, line in enumerate(lines(label, width)):
        result.append(f'<text x="{x:.1f}" y="{y + index * leading:.1f}" text-anchor="{anchor}" fill="{color}" font-size="{size}" font-weight="{weight}">{escape(line)}</text>')
    return "".join(result)


def arrow_path(points, color, width=2.2):
    d = "M " + " L ".join(f"{x:.1f} {y:.1f}" for x, y in points)
    (x0, y0), (x1, y1) = points[-2:]
    if abs(x1 - x0) >= abs(y1 - y0):
        sign = 1 if x1 > x0 else -1
        base = [(x1 - sign * 10, y1 - 5), (x1 - sign * 10, y1 + 5)]
    else:
        sign = 1 if y1 > y0 else -1
        base = [(x1 - 5, y1 - sign * 10), (x1 + 5, y1 - sign * 10)]
    triangle = " ".join(f"{x:.1f},{y:.1f}" for x, y in [(x1, y1), *base])
    return (f'<path d="{d}" fill="none" stroke="{color}" stroke-width="{width}" stroke-linejoin="miter" />'
            f'<polygon points="{triangle}" fill="{color}" />')


def chip(x, y, label, color, background, anchor="middle", width=22):
    wrapped = lines(label, width)
    w = max(45, max(len(s) for s in wrapped) * 8 + 16)
    h = len(wrapped) * 18 + 10
    left = x - w / 2 if anchor == "middle" else (x - w if anchor == "end" else x)
    top = y - 16
    result = f'<rect x="{left:.1f}" y="{top:.1f}" width="{w:.1f}" height="{h:.1f}" rx="4" fill="{background}" />'
    tx = x if anchor == "middle" else left + 8
    ta = "middle" if anchor == "middle" else "start"
    return result + text(tx, y, label, color, 14, ta, width=width, leading=18)


def slot(box, side, index, total):
    x, y, w, h = box
    fraction = (index + 1) / (total + 1)
    if side == "input":
        return (x, y + h * fraction)
    if side == "output":
        return (x + w, y + h * fraction)
    if side == "control":
        return (x + w * fraction, y)
    return (x + w * fraction, y + h)


def render_diagram(model, diagram):
    theme = {**DEFAULT_THEME, **model.get("theme", {}), **diagram.get("theme", {})}
    blocks = diagram["blocks"]
    context = diagram["id"] == "A-0"
    if context:
        max_vertical = max(sum(a["kind"] == kind for a in diagram["arrows"])
                           for kind in ("input", "output"))
        max_horizontal = max(sum(a["kind"] == kind for a in diagram["arrows"])
                             for kind in ("control", "mechanism"))
        bw = max(360, 155 * max_horizontal)
        bh = max(180, 45 * max_vertical)
        coords = {"A0": (300, 270, bw, bh)}
        canvas_w = bw + 650
        canvas_h = 270 + bh + 200
    else:
        bw, bh, gap_x, step_y = 250, 115, 125, 165
        coords = {block["id"]: (250 + i * (bw + gap_x), 240 + i * step_y, bw, bh)
                  for i, block in enumerate(blocks)}
        canvas_w = max(x + w for x, y, w, h in coords.values()) + 310
        canvas_h = max(y + h for x, y, w, h in coords.values()) + 190
    svg = [f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {canvas_w} {canvas_h}" role="img" aria-labelledby="title desc">',
           f'<title id="title">{escape(diagram["title"])}</title>',
           f'<desc id="desc">IDEF0 {escape(diagram["id"])}: {escape(model["title"])}</desc>',
           '<style>text{font-family:Arial,Helvetica,sans-serif}a{text-decoration:none}</style>',
           f'<rect width="{canvas_w}" height="{canvas_h}" fill="{theme["background"]}" />',
           f'<rect x="20" y="20" width="{canvas_w - 40}" height="{canvas_h - 40}" fill="none" stroke="{theme["frame"]}" />',
           text(50, 63, diagram["title"], theme["title"], 27, "start", "700", width=90),
           text(50, 93, f'IDEF0 · {diagram["id"]} · {model["title"]}', theme["subtitle"], 15, "start", width=100),
           f'<path d="M 50 115 L {canvas_w - 50} 115" stroke="{theme["frame"]}" />']
    counts = {}
    for block in blocks:
        bid = block["id"]
        counts[bid] = {side: [] for side in ("input", "control", "output", "mechanism")}
    for arrow in diagram["arrows"]:
        kind = arrow["kind"]
        if kind in INCOMING:
            counts[arrow["to"]][kind].append(arrow["id"])
        elif kind == "output":
            counts[arrow["from"]]["output"].append(arrow["id"])
        else:
            counts[arrow["from"]]["output"].append(arrow["id"])
            counts[arrow["to"]][arrow.get("to_kind", "input")].append(arrow["id"])
    def port(bid, side, aid):
        members = counts[bid][side]
        return slot(coords[bid], side, members.index(aid), len(members))
    paths, boxes_svg, labels = [], [], []
    for block in blocks:
        x, y, w, h = coords[block["id"]]
        style = block.get("style", {})
        fill = style.get("fill", theme["block_fill"])
        stroke = style.get("stroke", theme["block_stroke"])
        body = [f'<rect x="{x}" y="{y}" width="{w}" height="{h}" fill="{fill}" stroke="{stroke}" stroke-width="2.3" />',
                text(x + w / 2, y + h / 2 - (len(lines(block["label"], 22)) - 1) * 11,
                     block["label"], style.get("text", theme["block_text"]), 18, "middle", "700", width=22, leading=22),
                text(x + w - 13, y + h - 12, "0" if context else block["id"][len(diagram["id"]):],
                     style.get("number", theme["block_number"]), 15, "end", "700")]
        shape = "".join(body)
        if block.get("child"):
            shape = f'<a href="{escape(block["child"])}.svg">{shape}</a>'
        boxes_svg.append(shape)
    lane = 0
    for arrow in diagram["arrows"]:
        kind, aid = arrow["kind"], arrow["id"]
        style = arrow.get("style", {})
        color = style.get("stroke", theme["arrow"])
        label_color = style.get("text", theme["arrow_text"])
        if kind == "input":
            end = port(arrow["to"], "input", aid)
            start = (45, end[1])
            points = [start, end]
            labels.append(chip(55, end[1] - 14, arrow["label"], label_color, theme["background"], "start", 21))
        elif kind == "control":
            end = port(arrow["to"], "control", aid)
            start = (end[0], end[1] - 58)
            points = [start, end]
            labels.append(chip(end[0], end[1] - 87, arrow["label"], label_color, theme["background"], width=20))
        elif kind == "mechanism":
            end = port(arrow["to"], "mechanism", aid)
            start = (end[0], end[1] + 58)
            points = [start, end]
            labels.append(chip(end[0], end[1] + 75, arrow["label"], label_color, theme["background"], width=20))
        elif kind == "output":
            start = port(arrow["from"], "output", aid)
            end = (canvas_w - 45, start[1])
            points = [start, end]
            labels.append(chip(canvas_w - 265, start[1] - 16, arrow["label"], label_color, theme["background"], "start", 26))
        else:
            start = port(arrow["from"], "output", aid)
            target_side = arrow.get("to_kind", "input")
            end = port(arrow["to"], target_side, aid)
            source_index = next(i for i, b in enumerate(blocks) if b["id"] == arrow["from"])
            target_index = next(i for i, b in enumerate(blocks) if b["id"] == arrow["to"])
            if target_index == source_index + 1 and target_side == "input":
                mid_x = (start[0] + end[0]) / 2
                points = [start, (mid_x, start[1]), (mid_x, end[1]), end]
                labels.append(chip(mid_x, (start[1] + end[1]) / 2 - 9,
                                   arrow["label"], label_color, theme["background"], width=17))
            else:
                lane_y = max(y + h for x, y, w, h in coords.values()) + 80 + lane * 40
                lane += 1
                sx = start[0] + 45
                ex = end[0] - 45 if target_side == "input" else end[0]
                points = [start, (sx, start[1]), (sx, lane_y), (ex, lane_y)]
                if target_side == "input":
                    points += [(ex, end[1]), end]
                else:
                    points += [(ex, end[1] - 30), end]
                labels.append(chip((sx + ex) / 2, lane_y - 15, arrow["label"], label_color,
                                   theme["background"], width=24))
        paths.append(arrow_path(points, color))
    svg += paths + boxes_svg + labels
    svg += [f'<path d="M 50 {canvas_h - 60} L {canvas_w - 50} {canvas_h - 60}" stroke="{theme["frame"]}" />',
            text(50, canvas_h - 32, model.get("viewpoint", ""), theme["subtitle"], 14, "start", width=95),
            text(canvas_w - 50, canvas_h - 32, diagram["id"], theme["title"], 15, "end", "700"),
            '</svg>']
    return "\n".join(svg) + "\n"


def output_dir_for(path, model, override=None):
    if override:
        return Path(override)
    name = model.get("output_dir")
    if name is None:
        name = path.name.removesuffix(".idef.json") + "-svg"
    if not isinstance(name, str) or not name or Path(name).is_absolute() or ".." in Path(name).parts:
        fail("model.output_dir", "must be a relative directory below the model file")
    return path.parent / name


def render_file(path, output_dir=None, check=False):
    path = Path(path)
    model = json.loads(path.read_text(encoding="utf-8"))
    by_id = validate(model)
    destination = output_dir_for(path, model, output_dir)
    expected = {f"{did}.svg": render_diagram(model, diagram)
                for did, diagram in by_id.items()}
    manifest_name = ".idef-svg-manifest.json"
    manifest = json.dumps({"source": path.name, "files": sorted(expected)}, ensure_ascii=False, indent=2) + "\n"
    old_files = set()
    old_manifest = destination / manifest_name
    if old_manifest.exists():
        old_files = set(json.loads(old_manifest.read_text(encoding="utf-8")).get("files", []))
    if check:
        stale = [name for name, content in expected.items()
                 if not (destination / name).exists() or (destination / name).read_text(encoding="utf-8") != content]
        if not old_manifest.exists() or old_manifest.read_text(encoding="utf-8") != manifest or old_files - set(expected):
            stale.append(manifest_name)
        if stale:
            fail(str(path), "generated files are stale: " + ", ".join(stale))
    else:
        destination.mkdir(parents=True, exist_ok=True)
        for name in old_files - set(expected):
            if Path(name).name != name or not name.endswith(".svg"):
                fail(str(old_manifest), f"unsafe stale filename {name!r}")
            (destination / name).unlink(missing_ok=True)
        for name, content in expected.items():
            target = destination / name
            if not target.exists() or target.read_text(encoding="utf-8") != content:
                target.write_text(content, encoding="utf-8")
        if not old_manifest.exists() or old_manifest.read_text(encoding="utf-8") != manifest:
            old_manifest.write_text(manifest, encoding="utf-8")
    return len(expected), destination


def render_all(root, check=False):
    paths = sorted(Path(root).rglob("*.idef.json"))
    total = 0
    for path in paths:
        count, destination = render_file(path, check=check)
        total += count
        print(f"{path}: {count} diagram(s) -> {destination}")
    return total


def main(argv=None):
    parser = argparse.ArgumentParser(prog="idef-svg", description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    one = sub.add_parser("render", help="render one *.idef.json model")
    one.add_argument("model", type=Path)
    one.add_argument("--output-dir", type=Path)
    one.add_argument("--check", action="store_true", help="verify generated files without writing")
    all_models = sub.add_parser("render-all", help="render every *.idef.json below a directory")
    all_models.add_argument("directory", type=Path)
    all_models.add_argument("--check", action="store_true")
    args = parser.parse_args(argv)
    try:
        if args.command == "render":
            count, destination = render_file(args.model, args.output_dir, args.check)
            print(f"{count} diagram(s) -> {destination}")
        else:
            print(f"Total: {render_all(args.directory, args.check)} diagram(s)")
    except (ModelError, OSError, json.JSONDecodeError) as error:
        parser.exit(1, f"idef-svg: {error}\n")


if __name__ == "__main__":
    main()
