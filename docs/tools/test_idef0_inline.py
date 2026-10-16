"""Checks for IDEF0 diagrams authored inside Markdown."""

import unittest
import re
import tempfile
from pathlib import Path
from types import SimpleNamespace

import idef0_inline
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
@view focus `Обработать заявку`
@view decompose `Обработать заявку`
`Обработать заявку` receives `Заявка`
`Обработать заявку` produces `Решение`
`Обработать заявку` is composed of `Проверить заявку`
`Проверить заявку` receives `Заявка`
`Проверить заявку` produces `Решение`
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
        markdown = next(example for example in examples if "@view focus `Manage Hiring`" in example)
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

    def test_quoted_names_and_english_predicates(self):
        markdown = """```idef0
@view decompose "Обработать заявку"
@block `Проверить заявку` | fill=#EAF6EE
`Обработать заявку` receives "Заявка"
`Обработать заявку` produces `Решение`
`Обработать заявку` is composed of `Проверить заявку`
`Проверить заявку` receives `Заявка`
`Проверить заявку` produces `Решение`
```\n"""
        result = render_inline_diagrams(markdown, "example.md")
        self.assertIn("data-idef0-block='Проверить заявку'", result)
        self.assertIn("fill='#EAF6EE'", result)

    def test_example_inside_longer_fence_stays_literal(self):
        markdown = """````markdown
```idef0
Assess Candidate receives Application
```
````
"""
        self.assertEqual(render_inline_diagrams(markdown, "guide.md"), markdown)

    def test_boundary_error_identifies_page(self):
        model = """Manage Hiring receives Hiring Request
Manage Hiring produces Hiring Decision
Manage Hiring is composed of Screen Applications
Manage Hiring is composed of Prepare Shortlist
Screen Applications receives Hiring Request
Screen Applications requires Screening Tool
Screen Applications produces Shortlist
Prepare Shortlist receives Shortlist
Prepare Shortlist produces Hiring Decision"""
        with self.assertRaisesRegex(ValueError, "hiring.md:1: IDEF0: .*boundary mismatch"):
            render_inline_diagrams("```idef0\n" + model + "\n```\n", "hiring.md")

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
            self.assertEqual(export_markdown_diagrams(
                docs, on_error=lambda page, export, views, error: errors.append((page, export, views, error))), 1)
            self.assertTrue((docs / "good.svg").exists())
            self.assertEqual(len(errors), 1)
            self.assertEqual(errors[0][1], "bad.svg")
            self.assertIn("bad.md:3: IDEF0", str(errors[0][3]))
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

    def test_build_keeps_page_and_replaces_invalid_diagram(self):
        page = SimpleNamespace(file=SimpleNamespace(src_uri="project/demo.md"))
        config = SimpleNamespace(extra={"_idef0_errors": {
            "models": [{"source": "idef0-svg/models/bad.idef0", "message": "bad.idef0: line 2"}],
            "exports": [],
        }})
        markdown = """# Demo

![Broken model](../idef0-svg/models/bad.svg)

```idef0
@view wrong `Процесс`
`Процесс` receives `Вход`
```
"""
        result = idef0_inline.on_page_markdown(markdown, page=page, config=config, files=None)
        self.assertEqual(result.count('class="idef0-error"'), 3)
        self.assertNotIn("![Broken model]", result)
        self.assertIn("project/demo.md:6: IDEF0", result)

    def test_failed_export_image_is_replaced_but_code_example_is_kept(self):
        page = SimpleNamespace(file=SimpleNamespace(src_uri="project/demo.md"))
        config = SimpleNamespace(extra={"_idef0_errors": {
            "models": [],
            "exports": [{"target": "contributing/idef0/figure.svg", "message": "export failed"}],
        }})
        image = "![Nested view](../contributing/idef0/figure-2.svg)"
        markdown = f"# Demo\n\n{image}\n\n```markdown\n{image}\n```\n"
        result = idef0_inline.on_page_markdown(markdown, page=page, config=config, files=None)
        self.assertEqual(result.count('class="idef0-error"'), 2)
    def test_order_preserves_declaration_and_supports_heuristic(self):
        markdown_default = """```idef0
@view decompose `Pipeline`
`Pipeline` receives `Data`
`Pipeline` produces `Verdict`
`Pipeline` is composed of `Step Connect`
`Pipeline` is composed of `Step Code`
`Pipeline` is composed of `Step Verdict`
`Step Connect` receives `Data`
`Step Connect` produces `Session`
`Step Code` receives `Session`
`Step Code` produces `Code`
`Step Verdict` receives `Code`
`Step Verdict` produces `Verdict`
```
"""
        result_default = render_inline_diagrams(markdown_default, "order_test.md")
        pos_connect = result_default.find("data-idef0-block='Step Connect'")
        pos_code = result_default.find("data-idef0-block='Step Code'")
        pos_verdict = result_default.find("data-idef0-block='Step Verdict'")
        self.assertTrue(pos_connect < pos_code < pos_verdict)

    def test_branching_mechanisms_and_toggle(self):
        markdown_branched = """```idef0
@view decompose `Process`
`Process` receives `In`
`Process` requires `Team`
`Process` produces `Out`
`Process` is composed of `A`
`Process` is composed of `B`
`A` receives `In`
`A` requires `Team`
`A` produces `Mid`
`B` receives `Mid`
`B` requires `Team`
`B` produces `Out`
```
"""
        result_branched = render_inline_diagrams(markdown_branched, "branch_test.md")
        self.assertEqual(result_branched.count("data-idef0-arrow='Team'"), 1)
        self.assertEqual(result_branched.count("data-idef0-icom='M1'"), 1)

        markdown_unbranched = """```idef0
@branching none
@view decompose `Process`
`Process` receives `In`
`Process` requires `Team`
`Process` produces `Out`
`Process` is composed of `A`
`Process` is composed of `B`
`A` receives `In`
`A` requires `Team`
`A` produces `Mid`
`B` receives `Mid`
`B` requires `Team`
`B` produces `Out`
```
"""
        result_unbranched = render_inline_diagrams(markdown_unbranched, "branch_unbranched.md")
        self.assertEqual(result_unbranched.count("data-idef0-arrow='Team'"), 2)

    def test_corners_sharp_and_rounded(self):
        markdown_sharp = """```idef0
@corners sharp
@view decompose `Process`
`Process` receives `In`
`Process` produces `Out`
`Process` is composed of `A`
`Process` is composed of `B`
`A` receives `In`
`A` produces `Mid`
`B` receives `Mid`
`B` produces `Out`
```
"""
        result_sharp = render_inline_diagrams(markdown_sharp, "corners_sharp.md")
        self.assertNotIn(" C ", result_sharp)

        markdown_rounded = """```idef0
@corners rounded
@view decompose `Process`
`Process` receives `In`
`Process` produces `Out`
`Process` is composed of `A`
`Process` is composed of `B`
`A` receives `In`
`A` produces `Mid`
`B` receives `Mid`
`B` produces `Out`
```
"""
        result_rounded = render_inline_diagrams(markdown_rounded, "corners_rounded.md")
        self.assertIn(" C ", result_rounded)

    def test_crossings_arc_gap_none(self):
        # A model with a feedback loop crossing an external input or control
        markdown_base = """```idef0
@view decompose `Process`
`Process` receives `In`
`Process` respects `Rule`
`Process` produces `Out`
`Process` is composed of `A`
`Process` is composed of `B`
`A` receives `In`
`A` receives `Feedback`
`A` produces `Mid`
`B` receives `Mid`
`B` respects `Rule`
`B` produces `Out`
`B` produces `Feedback`
```
"""
        result_arc = render_inline_diagrams(markdown_base.replace("@view", "@crossings arc\n@view"), "crossing_arc.md")
        self.assertIn("<path", result_arc)
        self.assertIn("<svg", result_arc)

        result_gap = render_inline_diagrams(markdown_base.replace("@view", "@crossings gap\n@view"), "crossing_gap.md")
        self.assertIn("<path", result_gap)

        result_none = render_inline_diagrams(markdown_base.replace("@view", "@crossings none\n@view"), "crossing_none.md")
        self.assertIn("<svg", result_none)

    def test_branching_corners_match_project_standard_and_preserve_curves_through_crossings(self):
        markdown = """```idef0
@view decompose `Process`
@branching all
@crossings arc
`Process` receives `In`
`Process` requires `Team`
`Process` produces `Out`
`Process` is composed of `A`
`Process` is composed of `B`
`Process` is composed of `C`
`A` receives `In`
`A` requires `Team`
`A` produces `Mid1`
`B` receives `Mid1`
`B` requires `Team`
`B` produces `Mid2`
`C` receives `Mid2`
`C` requires `Team`
`C` produces `Out`
```
"""
        result = render_inline_diagrams(markdown, "branching_corners.md")
        self.assertIn("<svg", result)
        # Verify that cubic Bézier curves (C ...) are used for branching turns
        self.assertIn(" C ", result)
        # Verify that no broken diagonal lines occur (last target and turn points have proper curve)
        self.assertNotIn(" Q ", result)

    def test_icom_toggle_and_snug_alignment(self):
        markdown_on = """```idef0
@view decompose `Process`
@icom on
`Process` receives `In`
`Process` produces `Out`
`Process` is composed of `A`
`A` receives `In`
`A` produces `Out`
```
"""
        result_on = render_inline_diagrams(markdown_on, "icom_on.md")
        self.assertIn("class='idef0-icom'", result_on)

        markdown_off = """```idef0
@view decompose `Process`
@icom off
`Process` receives `In`
`Process` produces `Out`
`Process` is composed of `A`
`A` receives `In`
`A` produces `Out`
```
"""
        result_off = render_inline_diagrams(markdown_off, "icom_off.md")
        self.assertNotIn("class='idef0-icom'", result_off)
        # Verify metadata data-idef0-icom is still present on <g>
        self.assertIn("data-idef0-icom=", result_off)

    def test_label_collision_staggering_and_callouts(self):
        markdown_callout = """```idef0
@view decompose `Process`
@callouts on
`Process` receives `In`
`Process` requires `M1`
`Process` requires `M2`
`Process` requires `M3`
`Process` produces `Out`
`Process` is composed of `A`
`Process` is composed of `B`
`A` receives `In`
`A` requires `M1`
`A` requires `M2`
`A` requires `M3`
`B` requires `M1`
`B` requires `M2`
`B` requires `M3`
`B` produces `Out`
```
"""
        result_callout = render_inline_diagrams(markdown_callout, "callout_on.md")
        self.assertIn("<svg", result_callout)
        # When 3 branched mechanisms heavily overlap, the 3rd is pushed down to row 2 and receives a dashed callout line
        self.assertIn("stroke-dasharray='3,3'", result_callout)

        markdown_no_callout = markdown_callout.replace("@callouts on", "@callouts off")
        result_no_callout = render_inline_diagrams(markdown_no_callout, "callout_off.md")
        self.assertNotIn("stroke-dasharray='3,3'", result_no_callout)

    def test_custom_radius_directive(self):
        markdown_r6 = """```idef0
@view decompose `Process`
@radius 6
`Process` receives `In`
`Process` produces `Out`
`Process` is composed of `A`
`Process` is composed of `B`
`A` receives `In`
`A` produces `Mid`
`B` receives `Mid`
`B` produces `Out`
```
"""
        result_r6 = render_inline_diagrams(markdown_r6, "radius_6.md")
        self.assertIn("<svg", result_r6)
        self.assertIn(" C ", result_r6)

    def test_curve_rounding_direction_and_branch_turns(self):
        # Verify vertical-to-horizontal curve direction has no loop/hook
        markdown = """```idef0
@view decompose `Process`
@radius 8
`Process` receives `In`
`Process` requires `M`
`Process` produces `Out`
`Process` is composed of `A`
`Process` is composed of `B`
`Process` is composed of `C`
`A` receives `In`
`A` requires `M`
`A` produces `Mid1`
`B` receives `Mid1`
`B` requires `M`
`B` produces `Mid2`
`C` receives `Mid2`
`C` requires `M`
`C` produces `Out`
```
"""
        result = render_inline_diagrams(markdown, "curve_test.md")
        self.assertIn("<svg", result)

        # 1. Forward input curve (A -> B turn) must end smoothly at xc + r, without looping back
        # d contains: L <xc> <yc-r> C <xc> <yc-cp> <xc+cp> <yc> <xc+r> <yc> L ...
        curve_match = re.search(r"L\s+([-\d.]+)\s+([-\d.]+)\s+C\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+L\s+([-\d.]+)", result)
        self.assertIsNotNone(curve_match)
        xc, y_start = float(curve_match.group(1)), float(curve_match.group(2))
        cp1x, cp1y = float(curve_match.group(3)), float(curve_match.group(4))
        cp2x, cp2y = float(curve_match.group(5)), float(curve_match.group(6))
        end_x, end_y = float(curve_match.group(7)), float(curve_match.group(8))
        next_x = float(curve_match.group(9))

        # End x must be between xc and next_x (strictly advancing towards target)
        self.assertGreater(end_x, xc)
        self.assertLessEqual(end_x, next_x)

        # 2. In branched mechanism (M branches to B), branch curve must turn UPWARD (y decreases towards target)
        branch_match = re.search(r"M\s+([-\d.]+)\s+([-\d.]+)\s+C\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+L\s+([-\d.]+)\s+([-\d.]+)", result)
        self.assertIsNotNone(branch_match)
        b_bus_y = float(branch_match.group(2))
        b_end_y = float(branch_match.group(8))
        b_target_y = float(branch_match.group(10))
        # Turning UP means y decreases: bus_y > end_y >= target_y
        self.assertLess(b_end_y, b_bus_y)
        self.assertLessEqual(b_target_y, b_end_y)


if __name__ == "__main__":
    unittest.main()
