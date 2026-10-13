"""Keep automatic navigation complete during MkDocs dirty reloads.

MkDocs 1.6 recreates Pages on every build, but dirty mode skips reading and
rendering unchanged pages. Their titles are then None, and their HTML retains
the old menu. Read every source, reuse unchanged rendered Markdown, and rebuild
the inexpensive theme HTML for all pages. PlantUML runs only for changed pages.
"""

from __future__ import annotations

from mkdocs.structure.nav import _add_parent_links, _add_previous_and_next_links, _get_by_type
from mkdocs.structure.pages import Page

_rendered: dict[str, dict] = {}
_render_config: str | None = None
_diagnostics_signature: str | None = None
_FIELDS = ("content", "toc", "_title_from_render", "present_anchor_ids", "links_to_anchors")


def _first_src_uri(item) -> str:
    if getattr(item, "is_page", False) and getattr(item, "file", None) is not None:
        return item.file.src_uri
    for child in getattr(item, "children", None) or ():
        uri = _first_src_uri(child)
        if uri:
            return uri
    return ""


def _top_key(item) -> tuple[str, str]:
    uri = _first_src_uri(item).replace("\\", "/")
    top = uri.split("/", 1)[0] if uri else ""
    return uri, top


def _apply_section_titles(items, titles: dict[str, str]) -> None:
    for item in items:
        children = getattr(item, "children", None)
        if not children:
            continue
        _apply_section_titles(children, titles)
        slug = (item.title or "").strip().lower().replace(" ", "-")
        if slug in titles:
            item.title = titles[slug]
            continue
        for child in children:
            if getattr(child, "is_index", False) and child.title:
                item.title = child.title
                break


def on_config(config):
    global _render_config
    signature = repr((config.markdown_extensions, config.mdx_configs))
    if signature != _render_config:
        _rendered.clear()
        _render_config = signature
    return config


def on_nav(nav, *, config, files):
    global _diagnostics_signature
    signature = repr((config.get("extra") or {}).get("_idef0_errors", {}))
    if signature != _diagnostics_signature:
        _rendered.clear()
        _diagnostics_signature = signature
    for file in files.documentation_pages():
        page = file.page
        if page is None:
            continue
        page.read_source(config)
        cached = _rendered.get(file.src_uri)
        if not file.is_modified():
            if cached is not None and cached["markdown"] == page.markdown:
                for field, value in cached["fields"].items():
                    setattr(page, field, value)
            else:
                # Also recover if the output exists but this process has no cache.
                file.is_modified = lambda: True

    nav_extra = (config.get("extra") or {}).get("navigation") or {}
    raw_titles = nav_extra.get("titles") or {}
    titles = {str(k).strip().lower().replace(" ", "-"): str(v) for k, v in raw_titles.items()}
    _apply_section_titles(nav.items, titles)

    # Unwrap top-level single-page sections so they render as regular links instead of empty section headers.
    unwrapped = []
    for item in nav.items:
        children = getattr(item, "children", None)
        if children and len(children) == 1 and getattr(children[0], "is_index", False):
            page = children[0]
            page.parent = None
            unwrapped.append(page)
        else:
            unwrapped.append(item)
    nav.items[:] = unwrapped

    order = [str(x).strip().strip("/") for x in (nav_extra.get("order") or [])]
    if order:
        order_index = {key: idx for idx, key in enumerate(order)}

        def sort_key(enumerated_item):
            idx, item = enumerated_item
            uri, top = _top_key(item)
            if uri in order_index:
                return (0, order_index[uri], idx)
            if top in order_index:
                return (0, order_index[top], idx)
            return (1, idx, idx)

        nav.items[:] = [item for _, item in sorted(enumerate(nav.items), key=sort_key)]

    for item in nav.items:
        item.parent = None
    _add_parent_links(nav.items)
    nav.pages[:] = _get_by_type(nav.items, Page)
    _add_previous_and_next_links(nav.pages)
    return nav


def on_env(env, *, config, files):
    # This event runs AFTER Markdown population and BEFORE theme rendering.
    # Force only the latter for unchanged pages, not expensive diagram rendering.
    current = {}
    for file in files.documentation_pages():
        page = file.page
        if page is None or page.content is None:
            continue
        current[file.src_uri] = {
            "markdown": page.markdown,
            "fields": {field: getattr(page, field) for field in _FIELDS if hasattr(page, field)},
        }
        file.is_modified = lambda: True
    _rendered.clear()
    _rendered.update(current)
    return env
