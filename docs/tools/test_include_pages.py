import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace

from include_pages import number_sections, on_page_markdown


class IncludePagesTest(unittest.TestCase):
    def test_excerpt_includes_only_marked_passages_and_rebases_their_links(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / 'model' / 'index.md'
            source.parent.mkdir()
            source.write_text(
                '# Deployment\n\n<!-- excerpt -->\n\n![Pods](pods.svg)\n\n<!-- /excerpt -->\n'
                '### Technical details\n\n<!-- excerpt -->\n\n![Data](data.svg)\n\n<!-- /excerpt -->\n',
                encoding='utf-8',
            )
            page = SimpleNamespace(file=SimpleNamespace(src_uri='project/index.md'))
            result = on_page_markdown(
                '<!-- include-page: model/index.md excerpt -->',
                page=page, config={'docs_dir': directory}, files=None,
            )
            self.assertEqual(result.strip(), '![Pods](../model/pods.svg)\n\n![Data](../model/data.svg)')
            source.write_text('<!-- excerpt -->\nMissing end\n', encoding='utf-8')
            with self.assertRaisesRegex(ValueError, 'unclosed excerpt'):
                on_page_markdown(
                    '<!-- include-page: model/index.md excerpt -->',
                    page=page, config={'docs_dir': directory}, files=None,
                )

    def test_nested_page_rebases_images_and_local_anchors(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / 'architecture' / 'context(L1)' / 'index.md'
            source.parent.mkdir(parents=True)
            source.write_text('# Контекст\n\n![Схема](diagram.svg)\n\n[Подробнее](#details)\n', encoding='utf-8')
            page = SimpleNamespace(file=SimpleNamespace(src_uri='project/section/index.md'))
            result = on_page_markdown(
                '<!-- include-page: architecture/context(L1)/index.md depth=2 -->',
                page=page, config={'docs_dir': str(root)}, files=None,
            )
            self.assertIn('### Контекст', result)
            self.assertIn('../../architecture/context%28L1%29/diagram.svg', result)
            self.assertIn('../../architecture/context%28L1%29/index.md#details', result)

    def test_numbering_preserves_fenced_code_and_numbers_captions(self):
        content = ('## Контекст\n### Схема\n\n```plantuml\n## не заголовок\n```\n\n'
                   '<p align="center">Рисунок - Схема</p>\n\n'
                   '<p align="right">Таблица - Компоненты</p>\n')
        result = number_sections(content, '4.1')
        self.assertIn('## 4.1.1. Контекст', result)
        self.assertIn('### 4.1.1.1. Схема', result)
        self.assertIn('```plantuml\n## не заголовок\n```', result)
        self.assertIn('Рисунок 4.1.1 - Схема', result)
        self.assertIn('Таблица 4.1.1 - Компоненты', result)


if __name__ == '__main__':
    unittest.main()
