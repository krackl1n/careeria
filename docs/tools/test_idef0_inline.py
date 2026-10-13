"""Checks for IDEF0 diagrams authored inside Markdown."""

import unittest
import re
import tempfile
from pathlib import Path

from idef0_inline import export_markdown_diagrams, render_inline_diagrams
from idef0_render import MODELS


class Idef0InlineTest(unittest.TestCase):
    def test_guide_diagrams_render_together(self):
        guide = (Path(__file__).resolve().parents[1] / "contributing" / "idef0" / "index.md")
        result = render_inline_diagrams(guide.read_text(encoding="utf-8"),
                                        "contributing/idef0/index.md")
        self.assertEqual(result.count('<figure class="idef0-inline">'), 4)

    def test_russian_example_exports_svg_and_wraps_long_titles(self):
        guide = (Path(__file__).resolve().parents[1] / "contributing" / "idef0" / "index.md")
        examples = re.findall(r"````markdown\n(```idef0\n.*?\n```)\n````",
                              guide.read_text(encoding="utf-8"), re.S)
        markdown = next(example for example in examples if "@export obrabotka-zayavki.svg" in example)
        with tempfile.TemporaryDirectory() as directory:
            docs = Path(directory)
            (docs / "example.md").write_text(markdown, encoding="utf-8")
            self.assertEqual(export_markdown_diagrams(docs), 1)
            svg = (docs / "obrabotka-zayavki.svg").read_text(encoding="utf-8")
            self.assertIn("data-idef0-block='Проверить опыт кандидата", svg)
            self.assertIn("<tspan", svg)
            self.assertEqual(export_markdown_diagrams(docs, check=True), 1)
            (docs / "example.md").write_text(markdown.replace("Правила отбора", "Новые правила отбора"),
                                             encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "stale or missing"):
                export_markdown_diagrams(docs, check=True)
            self.assertEqual(export_markdown_diagrams(docs), 1)
            self.assertIn("Новые правила отбора",
                          (docs / "obrabotka-zayavki.svg").read_text(encoding="utf-8"))

    def test_multiple_views_get_predictable_export_names(self):
        markdown = """```idef0
@export assessment.svg
@view focus Assess Candidate
@view decompose Assess Candidate
Assess Candidate receives Application
Assess Candidate produces Assessment
Assess Candidate is composed of Review Profile
Review Profile receives Application
Review Profile produces Assessment
```
"""
        with tempfile.TemporaryDirectory() as directory:
            docs = Path(directory)
            (docs / "example.md").write_text(markdown, encoding="utf-8")
            self.assertEqual(export_markdown_diagrams(docs), 1)
            self.assertTrue((docs / "assessment.svg").exists())
            self.assertTrue((docs / "assessment-2.svg").exists())

    def test_cyrillic_function_can_be_selected_for_a_nested_view(self):
        markdown = """```idef0
@view focus Обработать заявку
@view decompose Обработать заявку
Обработать заявку получает Заявка
Обработать заявку создаёт Решение
Обработать заявку состоит из Проверить заявку
Проверить заявку получает Заявка
Проверить заявку создаёт Решение
```
"""
        result = render_inline_diagrams(markdown, "russian.md")
        self.assertEqual(result.count('<figure class="idef0-inline">'), 2)
        self.assertIn("data-idef0-block='Проверить заявку'", result)

    def test_simple_diagram_is_rendered_from_markdown(self):
        markdown = """# Example

```idef0
Assess Candidate receives Application
Assess Candidate produces Assessment
Assess Candidate is composed of Review Profile
Assess Candidate is composed of Record Result
Review Profile receives Application
Review Profile produces Findings
Record Result receives Findings
Record Result produces Assessment
```
"""
        result = render_inline_diagrams(markdown, "example.md")
        self.assertIn('<figure class="idef0-inline">', result)
        self.assertIn("data-idef0-block='Review Profile'", result)
        self.assertNotIn("```idef0", result)

    def test_views_tunnels_and_colors_share_one_model(self):
        guide = (Path(__file__).resolve().parents[1] / "contributing" / "idef0" / "index.md")
        examples = re.findall(r"````markdown\n(```idef0\n.*?\n```)\n````",
                              guide.read_text(encoding="utf-8"), re.S)
        markdown = next(example for example in examples if "@view focus Manage Hiring" in example)
        result = render_inline_diagrams(markdown, "hiring.md")
        self.assertEqual(result.count('<figure class="idef0-inline">'), 3)
        self.assertIn("data-idef0-tunnel-marker='parent'", result)
        self.assertIn("data-idef0-tunnel-marker='child'", result)
        self.assertIn("stroke='#15803D'", result)
        self.assertIn("fill='#EAF6EE'", result)

    def test_leaf_function_uses_focus_automatically(self):
        markdown = """```idef0
Issue Invoice receives Order
Issue Invoice produces Invoice
```
"""
        result = render_inline_diagrams(markdown, "invoice.md")
        self.assertIn("data-idef0-block='Issue Invoice'", result)

    def test_example_inside_longer_fence_stays_literal(self):
        markdown = """````markdown
```idef0
Assess Candidate receives Application
```
````
"""
        self.assertEqual(render_inline_diagrams(markdown, "guide.md"), markdown)

    def test_boundary_error_identifies_page(self):
        model = (MODELS / "hiring.idef0").read_text(encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "hiring.md:1: IDEF0: .*boundary mismatch"):
            render_inline_diagrams("```idef0\n" + model + "```\n", "hiring.md")

    def test_preview_shows_error_in_place_and_renders_next_diagram(self):
        markdown = """# Example

```idef0
@view unknown Process
Process receives Input
```

```idef0
Process receives Input
Process produces Output
```
"""
        result = render_inline_diagrams(markdown, "example.md", show_errors=True)
        self.assertIn('class="idef0-error"', result)
        self.assertIn("example.md:4: IDEF0", result)
        self.assertEqual(result.count('<figure class="idef0-inline">'), 1)
        with self.assertRaisesRegex(ValueError, "example.md:4: IDEF0"):
            render_inline_diagrams(markdown, "example.md")

    def test_preview_continues_exporting_other_pages_after_error(self):
        with tempfile.TemporaryDirectory() as directory:
            docs = Path(directory)
            (docs / "bad.md").write_text("```idef0\n@export bad.svg\n@view unknown Process\n"
                                          "Process receives Input\n```\n", encoding="utf-8")
            (docs / "good.md").write_text("```idef0\n@export good.svg\n"
                                           "Process receives Input\nProcess produces Output\n```\n",
                                           encoding="utf-8")
            errors = []
            self.assertEqual(export_markdown_diagrams(docs, on_error=errors.append), 1)
            self.assertTrue((docs / "good.svg").exists())
            self.assertEqual(len(errors), 1)
            self.assertIn("bad.md:3: IDEF0", str(errors[0]))
            with self.assertRaisesRegex(ValueError, "bad.md:3: IDEF0"):
                export_markdown_diagrams(docs)

    def test_renderer_error_uses_actual_statement_line(self):
        markdown = """```idef0
@view focus Process
Process receives Input
Process invents Result
```\n"""
        with self.assertRaisesRegex(ValueError, "example.md:4: IDEF0: .*unknown predicate"):
            render_inline_diagrams(markdown, "example.md")


if __name__ == "__main__":
    unittest.main()
