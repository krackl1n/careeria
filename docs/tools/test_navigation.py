"""Regression tests against the installed MkDocs dirty-build implementation."""

import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from mkdocs.commands.build import build
from mkdocs.config import load_config
from mkdocs.structure.pages import Page


class NavigationTests(unittest.TestCase):
    def test_dirty_reload_updates_every_menu_without_rerendering_unchanged_markdown(self):
        hook = Path(__file__).with_name("navigation.py").resolve()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            sources = root / "docs" / "backend"
            sources.mkdir(parents=True)
            config_file = root / "mkdocs.yml"
            config_file.write_text(
                f"site_name: Test\ndocs_dir: docs\nsite_dir: site\n"
                f"hooks:\n  - {hook}\nplugins: []\ntheme:\n  name: material\n", encoding="utf-8"
            )
            alpha = sources / "alpha.md"
            beta = sources / "beta.md"
            alpha.write_text("# Alpha Service\n\nFirst page.", encoding="utf-8")
            beta.write_text("# Beta Service\n\nSecond page.", encoding="utf-8")
            rendered = []
            original_render = Page.render

            def record_render(page, config, files):
                rendered.append(page.file.src_uri)
                return original_render(page, config, files)

            def rebuild():
                rendered.clear()
                build(load_config(config_file=str(config_file)), dirty=True)

            def assert_menus(expected, absent=()):
                for name in expected:
                    html = (root / "site" / "backend" / name / "index.html").read_text()
                    self.assertNotIn(">None<", html)
                    for title in expected.values():
                        self.assertIn(title, html)
                    for title in absent:
                        self.assertNotIn(title, html)

            with patch.object(Page, "render", record_render):
                rebuild()
                self.assertEqual(len(rendered), 2)
                beta.write_text("# Renamed Service\n\nUpdated.", encoding="utf-8")
                rebuild()
                self.assertEqual(rendered, ["backend/beta.md"])
                assert_menus({"alpha": "Alpha Service", "beta": "Renamed Service"}, ["Beta Service"])

                gamma = sources / "gamma.md"
                gamma.write_text("# Gamma Service", encoding="utf-8")
                rebuild()
                self.assertEqual(rendered, ["backend/gamma.md"])
                assert_menus({"alpha": "Alpha Service", "beta": "Renamed Service", "gamma": "Gamma Service"})

                gamma.unlink()
                rebuild()
                self.assertEqual(rendered, [])
                assert_menus({"alpha": "Alpha Service", "beta": "Renamed Service"}, ["Gamma Service"])


if __name__ == "__main__":
    unittest.main()
