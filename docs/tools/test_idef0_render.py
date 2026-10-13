"""Smoke checks for the vendored IDEF0-SVG color adapter."""

import subprocess
import unittest
import xml.etree.ElementTree as ET

from idef0_render import MODELS, RENDERER, render_models

SVG = "{http://www.w3.org/2000/svg}"


class Idef0RenderTest(unittest.TestCase):
    def test_nested_view(self):
        self.assertEqual(render_models(check=True), 2)
        source = MODELS / "hiring-review-applications.svg"
        root = ET.parse(source).getroot()
        blocks = {group.get("data-idef0-block") for group in root.iter(f"{SVG}g")
                  if group.get("data-idef0-block")}
        self.assertEqual(blocks, {"Screen Applications", "Evaluate Candidates", "Prepare Shortlist"})

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


if __name__ == "__main__":
    unittest.main()
