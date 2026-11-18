"""Include Markdown pages while preserving their relative links and heading hierarchy."""

from pathlib import Path
import posixpath
import re
from urllib.parse import urlsplit, urlunsplit

INCLUDE = re.compile(r'^[ \t]*<!-- include-page: ([^ ]+) -->[ \t]*$', re.MULTILINE)
LINK = re.compile(r'(\]\()([^\s)]+)(\))')
HEADING = re.compile(r'^(#{1,5})(\s+.*)$')


def number_information_structure(content):
    """Number section 4.2 headings in the assembled document's order."""
    counts = [0, 0, 0, 0]
    figures = 0
    tables = 0
    lines = []
    fence = None
    for line in content.splitlines():
        marker = re.match(r'^\s*(`{3,}|~{3,})', line)
        if marker:
            char = marker.group(1)[0]
            fence = char if fence is None else (None if fence == char else fence)
        if fence is None and not marker:
            if line.startswith('<p align="center">Рисунок - '):
                figures += 1
                line = line.replace('Рисунок - ', f'Рисунок 4.2.{figures} - ', 1)
            elif line.startswith('<p align="right">Таблица - '):
                tables += 1
                line = line.replace('Таблица - ', f'Таблица 4.2.{tables} - ', 1)
            heading = re.match(r'^(#{2,5})\s+(.+)$', line)
            if heading:
                level = len(heading.group(1)) - 2
                counts[level] += 1
                counts[level + 1:] = [0] * (len(counts) - level - 1)
                number = '.'.join(['4', '2'] + [str(x) for x in counts[:level + 1]])
                line = f'{heading.group(1)} {number}. {heading.group(2)}'
        lines.append(line)
    return '\n'.join(lines) + '\n'


def on_nav(nav, *, config, files):
    # MkDocs dirty reload cannot otherwise see the included-page dependencies.
    for file in files.documentation_pages():
        if file.page and file.page.markdown and INCLUDE.search(file.page.markdown):
            file.is_modified = lambda: True
    return nav


def on_page_markdown(markdown, *, page, config, files):
    root = Path(config['docs_dir']).resolve()
    target = (root / page.file.src_uri).parent

    def include(match):
        source = (root / match.group(1)).resolve()
        if not source.is_relative_to(root) or not source.is_file():
            raise ValueError(f'Invalid included page: {match.group(1)}')
        content = source.read_text(encoding='utf-8')
        lines = []
        fence = None
        for line in content.splitlines():
            marker = re.match(r'^\s*(`{3,}|~{3,})', line)
            if marker:
                char = marker.group(1)[0]
                if fence is None:
                    fence = char
                elif fence == char:
                    fence = None
                lines.append(line)
                continue
            if fence is None:
                line = HEADING.sub(r'#\1\2', line)
                def rebase(m):
                    url = urlsplit(m.group(2))
                    if url.scheme or url.netloc or not url.path or url.path.startswith('/'):
                        return m.group(0)
                    absolute = (source.parent / url.path).resolve()
                    relative = posixpath.relpath(absolute, target)
                    return m.group(1) + urlunsplit(('', '', relative, url.query, url.fragment)) + m.group(3)
                line = LINK.sub(rebase, line)
            lines.append(line)
        return '\n'.join(lines).rstrip() + '\n\n'

    result = INCLUDE.sub(include, markdown)
    if page.file.src_uri == 'project/04-technology-infrastructure/02-information-structure/index.md':
        result = number_information_structure(result)
    return result
