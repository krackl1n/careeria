#!/usr/bin/env python3
"""Check the built documentation, including native PlantUML rendering and links."""
from html import unescape
from html.parser import HTMLParser
from pathlib import Path
import re
from urllib.parse import unquote, urlsplit

root = Path(__file__).resolve().parents[2]
site = root / '.site-docs'
docs = root / 'docs'
errors = []

class Links(HTMLParser):
    def __init__(self):
        super().__init__()
        self.links = []
    def handle_starttag(self, tag, attrs):
        for key, value in attrs:
            if key in ('href', 'src') and value:
                self.links.append(value)

count = 0
for source in docs.rglob('*.md'):
    if any(part.startswith('.') or part == 'node_modules' for part in source.relative_to(docs).parts):
        continue
    markdown = source.read_text()
    diagrams = len(re.findall(r'^```plantuml\s*$', markdown, re.M))
    if not diagrams:
        continue
    relative = source.relative_to(docs)
    target = site / relative.with_suffix('') / 'index.html'
    if not target.exists():
        errors.append(f'Missing diagram page: {relative}')
        continue
    html = unescape(target.read_text())
    svg = re.findall(r'<svg\b[^>]*data-diagram-type="[^"]+"[^>]*>[\s\S]*?</svg>', html)
    if len(svg) != diagrams:
        errors.append(f'{relative}: expected {diagrams} SVG, got {len(svg)}')
    for diagram in svg:
        if re.search(r'Syntax Error|An error has occurred|Bad sub name', diagram):
            errors.append(f'PlantUML rendering error: {relative}')
        if 'preserveAspectRatio="none"' in diagram:
            errors.append(f'Stretched SVG: {relative}')
    count += diagrams

for target in site.rglob('*.html'):
    parser = Links()
    parser.feed(target.read_text())
    for href in parser.links:
        url = urlsplit(href)
        if url.scheme or url.netloc or not url.path:
            continue
        destination = site / unquote(url.path).lstrip('/') if url.path.startswith('/') else target.parent / unquote(url.path)
        if not destination.exists():
            errors.append(f'{target.relative_to(site)}: broken link {href}')

context = (site / 'architecture/c4-plant-uml/plantuml/context/system-context/index.html').read_text()
if '/architecture/c4-plant-uml/plantuml/container/full-system/' not in context:
    errors.append('Missing clickable context-to-container link')
for name in ['team', 'standalone']:
    group = site / f'architecture/c4-plant-uml/plantuml/integration/{name}'
    for page in group.glob('*/index.html'):
        html = unescape(page.read_text())
        if 'L2' in page.parent.name and 'интегрированная контейнерная диаграмма' not in html:
            errors.append(f'{page.relative_to(site)}: wrong integrated view')

if errors:
    raise SystemExit('\n'.join(sorted(set(errors))))
print(f'MkDocs: {count} диаграмм; страницы и внутренние ссылки проверены.')
