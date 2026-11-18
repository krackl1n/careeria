"""Include Markdown pages while preserving their relative links and heading hierarchy."""

from pathlib import Path
import posixpath
import re
from urllib.parse import quote, urlsplit, urlunsplit

INCLUDE = re.compile(
    r'^[ \t]*<!-- include-page: (\S+)(?: depth=([12]))?(?: (excerpt))?(?: (headings))? -->[ \t]*$', re.MULTILINE,
)
LINK = re.compile(r'(\]\()([^\s)]+)(\))')
HEADING = re.compile(r'^(#{1,5})(\s+.*)$')


def page_excerpt(content, source, headings=False):
    """Select the passages marked for inclusion, in their original order."""
    passages = []
    current = None
    title = None
    for line in content.splitlines():
        heading = re.match(r'^#{1,6}\s+(.+)$', line)
        if heading:
            title = heading.group(1).replace('—', '-')
        if line == '<!-- excerpt -->':
            if current is not None:
                raise ValueError(f'Nested excerpt in {source}')
            current = []
        elif line == '<!-- /excerpt -->':
            if not current:
                raise ValueError(f'Unexpected or empty excerpt in {source}')
            passage = '\n'.join(current).strip()
            if headings and title:
                passage = f'## {title} {{.unnumbered}}\n\n{passage}'
            passages.append(passage)
            current = None
        elif current is not None:
            current.append(line)
    if current is not None or not passages:
        raise ValueError(f'Missing or unclosed excerpt in {source}')
    return '\n\n'.join(passages)


def number_sections(content, prefix):
    """Number headings and captions in an assembled document's order."""
    counts = [0, 0, 0, 0, 0]
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
                line = line.replace('Рисунок - ', f'Рисунок {prefix}.{figures} - ', 1)
            elif line.startswith('<p align="right">Таблица - '):
                tables += 1
                line = line.replace('Таблица - ', f'Таблица {prefix}.{tables} - ', 1)
            heading = re.match(r'^(#{2,6})\s+(.+)$', line)
            if heading:
                if '{.unnumbered}' in line:
                    lines.append(line)
                    continue
                level = len(heading.group(1)) - 2
                counts[level] += 1
                counts[level + 1:] = [0] * (len(counts) - level - 1)
                number = '.'.join([prefix] + [str(x) for x in counts[:level + 1]])
                title = heading.group(2)
                if prefix == '4.1':
                    title = re.sub(r'^(?:\d+\.)+\s*', '', title)
                line = f'{heading.group(1)} {number}. {title}'
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
        if match.group(3):
            content = page_excerpt(content, source, headings=bool(match.group(4)))
        depth = int(match.group(2) or '1')
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
                def shift_heading(m):
                    return '#' * min(6, len(m.group(1)) + depth) + m.group(2)
                line = HEADING.sub(shift_heading, line)
                def rebase(m):
                    url = urlsplit(m.group(2))
                    if url.scheme or url.netloc or url.path.startswith('/'):
                        return m.group(0)
                    if not url.path and not url.fragment:
                        return m.group(0)
                    absolute = (source.parent / url.path).resolve() if url.path else source
                    relative = quote(posixpath.relpath(absolute, target), safe='/%-._~')
                    return m.group(1) + urlunsplit(('', '', relative, url.query, url.fragment)) + m.group(3)
                line = LINK.sub(rebase, line)
            lines.append(line)
        return '\n'.join(lines).rstrip() + '\n\n'

    result = INCLUDE.sub(include, markdown)
    if page.file.src_uri == 'project/04-technology-infrastructure/02-information-structure/index.md':
        result = number_sections(result, '4.2')
    elif page.file.src_uri == 'project/04-technology-infrastructure/01-c4-architecture/index.md':
        result = number_sections(result, '4.1')
    return result
