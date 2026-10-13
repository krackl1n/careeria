"""Smoke checks for the vendored IDEF0-SVG color adapter."""

import subprocess
import json
import tempfile
import unittest
import xml.etree.ElementTree as ET
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

import idef0_inline
import idef0_render
from idef0_render import MODELS, RENDERER, render_models

SVG = "{http://www.w3.org/2000/svg}"


class Idef0RenderTest(unittest.TestCase):
    def test_new_model_and_changed_views_are_discovered(self):
        with tempfile.TemporaryDirectory() as directory:
            models = Path(directory)
            source = models / "candidate-assessment.idef0"
            source.write_text(
                "Assess Candidate receives Application\n"
                "Assess Candidate produces Assessment\n"
                "Assess Candidate is composed of Review Profile\n"
                "Assess Candidate is composed of Record Result\n"
                "Review Profile receives Application\n"
                "Review Profile produces Findings\n"
                "Record Result receives Findings\n"
                "Record Result produces Assessment\n",
                encoding="utf-8",
            )
            with patch.object(idef0_render, "MODELS", models):
                self.assertEqual(render_models(), 1)
                output = models / "candidate-assessment.svg"
                self.assertTrue(output.exists())
                original = output.read_text(encoding="utf-8")

                style = models / "candidate-assessment.idef0.json"
                style.write_text(json.dumps({
                    "mode": "decompose", "process": "Assess Candidate",
                    "views": [{"name": "context", "mode": "focus", "process": "Assess Candidate"}],
                }), encoding="utf-8")
                self.assertEqual(render_models(), 2)
                self.assertTrue((models / "candidate-assessment-context.svg").exists())

                source.write_text(source.read_text(encoding="utf-8").replace(
                    "Review Profile", "Evaluate Profile"), encoding="utf-8")
                self.assertEqual(render_models(), 2)
                self.assertNotEqual(output.read_text(encoding="utf-8"), original)
                self.assertIn("Evaluate Profile", output.read_text(encoding="utf-8"))

    def test_nested_view(self):
        self.assertEqual(render_models(check=True), 7)
        source = MODELS / "hiring-review-applications.svg"
        root = ET.parse(source).getroot()
        blocks = {group.get("data-idef0-block") for group in root.iter(f"{SVG}g")
                  if group.get("data-idef0-block")}
        self.assertEqual(blocks, {"Screen Applications", "Evaluate Candidates", "Prepare Shortlist"})

    def test_boundary_codes_and_tunnels(self):
        parent = ET.parse(MODELS / "hiring.svg").getroot()
        child = ET.parse(MODELS / "hiring-review-applications.svg").getroot()
        parent_tunnels = [group for group in parent.iter(f"{SVG}g")
                          if group.get("data-idef0-tunnel-parent") == "Review Applications"]
        child_tunnels = [group for group in child.iter(f"{SVG}g")
                         if group.get("data-idef0-tunnel") == "child"]
        codes = {group.get("data-idef0-icom") for group in child.iter(f"{SVG}g")
                 if group.get("data-idef0-icom")}
        self.assertTrue(any(group.get("data-idef0-arrow") == "Hiring Policy"
                            for group in parent_tunnels))
        self.assertEqual(sum(group.get("data-idef0-tunnel-marker") == "parent"
                             for group in parent.iter(f"{SVG}g")), 1)
        self.assertEqual([group.get("data-idef0-arrow") for group in child_tunnels],
                         ["Screening Tool"])
        self.assertEqual(sum(group.get("data-idef0-tunnel-marker") == "child"
                             for group in child.iter(f"{SVG}g")), 1)
        self.assertEqual(codes, {"I1", "I2", "M1", "O1"})

    def test_untunneled_model_balances(self):
        root = ET.parse(MODELS / "vacancy-publication-prepare-vacancy.svg").getroot()
        self.assertFalse(any(group.get("data-idef0-tunnel")
                             or group.get("data-idef0-tunnel-parent")
                             for group in root.iter(f"{SVG}g")))
        codes = {group.get("data-idef0-icom") for group in root.iter(f"{SVG}g")
                 if group.get("data-idef0-icom")}
        self.assertEqual(codes, {"I1", "M1", "O1"})

    def test_unmarked_boundary_fails(self):
        source = MODELS / "hiring.idef0"
        style = json.loads(source.with_suffix(".idef0.json").read_text())
        style.pop("boundaries")
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "style.json"
            path.write_text(json.dumps(style))
            result = subprocess.run(
                ["ruby", str(RENDERER), "--input", str(source), "--style", str(path),
                 "--process", "Review Applications"],
                capture_output=True, text=True, check=False,
            )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("boundary mismatch", result.stderr)

    def test_per_element_colors(self):
        source = MODELS / "hiring.idef0"
        result = subprocess.run(
            ["ruby", str(RENDERER), "--input", str(source),
             "--style", str(source.with_suffix(".idef0.json")),
             "--process", "Manage Hiring"],
            capture_output=True, text=True, check=True,
        )
        root = ET.fromstring(result.stdout)
        blocks = {group.get("data-idef0-block"): group for group in root.iter(f"{SVG}g")
                  if group.get("data-idef0-block")}
        arrows = [group for group in root.iter(f"{SVG}g")
                  if group.get("data-idef0-arrow") == "Hiring Decision"]
        self.assertEqual(blocks["Make Decision"].find(f"{SVG}rect").get("fill"), "#EAF6EE")
        self.assertEqual(blocks["Make Decision"].find(f"{SVG}text").get("fill"), "#14532D")
        self.assertTrue(arrows)
        for group in arrows:
            self.assertEqual(group.find(f"{SVG}polygon").get("fill"), "#15803D")
            self.assertEqual(group.find(f"{SVG}text").get("fill"), "#14532D")

    def test_preview_reports_bad_model_and_renders_valid_one(self):
        with tempfile.TemporaryDirectory() as directory:
            models = Path(directory)
            (models / "bad.idef0").write_text("Bad Process invents Input\n", encoding="utf-8")
            (models / "good.idef0").write_text(
                "Good Process receives Input\n"
                "Good Process produces Output\n"
                "Good Process is composed of Do Work\n"
                "Do Work receives Input\n"
                "Do Work produces Output\n", encoding="utf-8")
            errors = []
            with patch.object(idef0_render, "MODELS", models):
                self.assertEqual(render_models(on_error=lambda source, error: errors.append((source, error))), 1)
                self.assertTrue((models / "good.svg").exists())
                self.assertEqual(len(errors), 1)
                self.assertEqual(errors[0][0].name, "bad.idef0")
                with self.assertRaises(RuntimeError):
                    render_models()

    def test_preview_model_error_is_visible_on_page_and_clears_after_fix(self):
        with tempfile.TemporaryDirectory() as directory:
            models = Path(directory)
            source = models / "bad.idef0"
            source.write_text("Bad Process invents Input\n", encoding="utf-8")
            config = SimpleNamespace(extra={})
            page = SimpleNamespace(file=SimpleNamespace(src_uri="index.md"))
            with patch.object(idef0_render, "MODELS", models):
                idef0_render.on_pre_build(config)
                self.assertIn("Ошибка IDEF0", (models / "bad.svg").read_text(encoding="utf-8"))
                output = idef0_inline.on_page_markdown("# Page\n", page=page,
                                                        config=config, files=None)
                self.assertIn('class="idef0-error"', output)
                self.assertIn("bad.idef0", output)
                source.unlink()
                idef0_render.on_pre_build(config)
                output = idef0_inline.on_page_markdown("# Page\n", page=page,
                                                        config=config, files=None)
                self.assertNotIn('class="idef0-error"', output)


if __name__ == "__main__":
    unittest.main()
