import subprocess
import tempfile
import unittest
from pathlib import Path

from sync_release_to_dev import sync_release_commit


def git(cwd, *args):
    return subprocess.check_output(["git", *args], cwd=cwd, text=True).strip()


class ReleaseMetadataSyncTest(unittest.TestCase):
    """Exercise the sync against a throwaway origin/main/dev triple."""

    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        root = Path(temp.name)
        bare = root / "origin.git"
        self.repo = root / "repo"
        subprocess.check_call(["git", "init", "--bare", str(bare)], stdout=subprocess.DEVNULL)
        subprocess.check_call(["git", "init", "-b", "main", str(self.repo)], stdout=subprocess.DEVNULL)
        git(self.repo, "config", "user.name", "Release test")
        git(self.repo, "config", "user.email", "release@example.invalid")
        git(self.repo, "remote", "add", "origin", str(bare))

        (self.repo / "VERSION").write_text("1.0.0\n", encoding="utf-8")
        (self.repo / "CHANGELOG.md").write_text("# Changelog\n", encoding="utf-8")
        git(self.repo, "add", ".")
        git(self.repo, "commit", "-m", "chore: initial state")
        git(self.repo, "push", "-u", "origin", "main")
        git(self.repo, "branch", "dev", "main")
        git(self.repo, "push", "origin", "dev")

    def create_release(self):
        """Commit and push the release metadata commit the workflow would create."""
        (self.repo / "VERSION").write_text("1.1.0\n", encoding="utf-8")
        (self.repo / "CHANGELOG.md").write_text("# Changelog\n\n## v1.1.0\n", encoding="utf-8")
        git(self.repo, "add", "VERSION", "CHANGELOG.md")
        git(self.repo, "commit", "-m", "chore(release): v1.1.0")
        release_commit = git(self.repo, "rev-parse", "HEAD")
        git(self.repo, "push", "origin", "main")
        return release_commit

    def commit_on(self, branch, path, message):
        """Commit one file on branch, push it, and return the new branch tip."""
        git(self.repo, "checkout", branch)
        (self.repo / path).write_text(f"{message}\n", encoding="utf-8")
        git(self.repo, "add", path)
        git(self.repo, "commit", "-m", message)
        git(self.repo, "push", "origin", branch)
        return git(self.repo, "rev-parse", "HEAD")

    def test_fast_forwards_dev_to_the_release_commit(self):
        release_commit = self.create_release()
        head_before = git(self.repo, "rev-parse", "HEAD")

        self.assertTrue(sync_release_commit(str(self.repo), release_commit))
        self.assertEqual(release_commit, git(self.repo, "rev-parse", "refs/remotes/origin/dev"))
        self.assertEqual(head_before, git(self.repo, "rev-parse", "HEAD"))
        self.assertEqual("main", git(self.repo, "rev-parse", "--abbrev-ref", "HEAD"))
        subjects = git(self.repo, "log", "--format=%s", "refs/remotes/origin/dev").splitlines()
        self.assertEqual(1, subjects.count("chore(release): v1.1.0"))
        self.assertFalse(sync_release_commit(str(self.repo), release_commit))

    def test_replays_only_release_metadata_onto_advanced_dev_and_is_idempotent(self):
        release_commit = self.create_release()
        self.commit_on("main", "hotfix.txt", "fix: hotfix landed on main only")
        dev_tip = self.commit_on("dev", "feature.txt", "feat: continue development")
        # CI jobs that only republish a tag never configure a committer identity.
        git(self.repo, "config", "--unset", "user.name")
        git(self.repo, "config", "--unset", "user.email")

        self.assertTrue(sync_release_commit(str(self.repo), release_commit))
        synced_tip = git(self.repo, "rev-parse", "refs/remotes/origin/dev")
        self.assertEqual("github-actions[bot]", git(self.repo, "log", "-1", "--format=%an", synced_tip))
        self.assertEqual("1.1.0", git(self.repo, "show", f"{synced_tip}:VERSION"))
        self.assertEqual("chore(release): v1.1.0", git(self.repo, "log", "-1", "--format=%s", synced_tip))
        self.assertEqual(dev_tip, git(self.repo, "rev-parse", f"{synced_tip}^"))
        self.assertTrue(git(self.repo, "ls-tree", "--name-only", synced_tip, "feature.txt"))
        self.assertEqual("", git(self.repo, "ls-tree", "--name-only", synced_tip, "hotfix.txt"))
        self.assertEqual(dev_tip, git(self.repo, "rev-parse", "dev"))

        self.assertFalse(sync_release_commit(str(self.repo), release_commit))
        self.assertEqual(synced_tip, git(self.repo, "rev-parse", "refs/remotes/origin/dev"))

    def test_reports_an_unreadable_release_commit_instead_of_claiming_dev_is_synced(self):
        self.create_release()
        with self.assertRaises(RuntimeError):
            sync_release_commit(str(self.repo), "0" * 40)


if __name__ == "__main__":
    unittest.main()
