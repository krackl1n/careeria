"""Keep automatic navigation complete during MkDocs dirty reloads.

MkDocs 1.6 recreates Pages on every build, but dirty mode skips reading and
rendering unchanged pages. Their titles are then None, and their HTML retains
the old menu. Read every source, reuse unchanged rendered Markdown, and rebuild
the inexpensive theme HTML for all pages. PlantUML runs only for changed pages.
"""

from __future__ import annotations

_rendered: dict[str, dict] = {}
_render_config: str | None = None
_FIELDS = ("content", "toc", "_title_from_render", "present_anchor_ids", "links_to_anchors")


def on_config(config):
    global _render_config
    signature = repr((config.markdown_extensions, config.mdx_configs))
    if signature != _render_config:
        _rendered.clear()
        _render_config = signature
    return config


def on_nav(nav, *, config, files):
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
