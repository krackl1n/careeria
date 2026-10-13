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

            def rebuild(diagnostics=None):
                rendered.clear()
                config = load_config(config_file=str(config_file))
                if diagnostics is not None:
                    config.extra["_idef0_errors"] = diagnostics
                build(config, dirty=True)

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

                rebuild({"models": ["broken.idef0: invalid model"]})
                self.assertEqual(set(rendered), {"backend/alpha.md", "backend/beta.md"})
                rebuild()
                self.assertEqual(set(rendered), {"backend/alpha.md", "backend/beta.md"})

    def test_section_titles_and_top_level_order(self):
        hook = Path(__file__).with_name("navigation.py").resolve()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            docs = root / "docs"
            (docs / "project" / "01-general").mkdir(parents=True)
            (docs / "deployment").mkdir(parents=True)
            (docs / "index.md").write_text("# Главная\n", encoding="utf-8")
            (docs / "project" / "index.md").write_text("# Описание проекта\n", encoding="utf-8")
            (docs / "project" / "01-general" / "index.md").write_text("# 1. Общие сведения\n", encoding="utf-8")
            (docs / "project" / "01-general" / "01-goal.md").write_text("# 1.1. Цель\n", encoding="utf-8")
            (docs / "deployment" / "index.md").write_text("# Развёртывание\n", encoding="utf-8")

            config_file = root / "mkdocs.yml"
            config_file.write_text(
                f"site_name: Test\ndocs_dir: docs\nsite_dir: site\n"
                f"hooks:\n  - {hook}\nplugins: []\n"
                f"theme:\n  name: material\n  features:\n    - navigation.expand\n    - navigation.indexes\n"
                f"extra:\n  navigation:\n    order:\n      - index.md\n      - project\n      - deployment\n",
                encoding="utf-8",
            )
            build(load_config(config_file=str(config_file)))
            html = (root / "site" / "index.html").read_text(encoding="utf-8")
            self.assertIn("1. Общие сведения", html)
            self.assertIn("1.1. Цель", html)
            self.assertIn("Развёртывание", html)
            self.assertIn("md-toggle--indeterminate", html)


if __name__ == "__main__":
    unittest.main()
