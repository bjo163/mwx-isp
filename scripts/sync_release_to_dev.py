#!/usr/bin/env python3
"""Sync an already-created release metadata commit onto the current dev tip."""

from __future__ import annotations

import argparse
import subprocess
import sys

# Files written by the release commit; only these are replayed onto a diverged dev.
RELEASE_PATHS = ("VERSION", "CHANGELOG.md")
# Release commits are authored by the workflow bot, so dev reuses that identity
# instead of relying on a repository-local git config being present.
BOT_IDENTITY = (
    "-c",
    "user.name=github-actions[bot]",
    "-c",
    "user.email=41898282+github-actions[bot]@users.noreply.github.com",
)


def git(repo: str, *args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(["git", *args], cwd=repo, text=True, capture_output=True)
    if check and result.returncode:
        raise RuntimeError(result.stderr.strip() or result.stdout.strip())
    return result


def is_ancestor(repo: str, ancestor: str, descendant: str) -> bool:
    return git(repo, "merge-base", "--is-ancestor", ancestor, descendant, check=False).returncode == 0


def release_subject(repo: str, release_commit: str) -> str:
    """Return the release commit subject, reused as the dev metadata commit message."""
    subject = git(repo, "show", "-s", "--format=%s", release_commit).stdout.strip()
    return subject or f"chore(release): metadata from {release_commit[:12]}"


def push_lost_race(pushed: subprocess.CompletedProcess[str]) -> bool:
    """Tell whether a rejected push only lost a race with another dev update."""
    output = pushed.stderr + pushed.stdout
    return "non-fast-forward" in output or "fetch first" in output


def sync_release_commit(repo: str, release_commit: str, attempts: int = 3) -> bool:
    """Fast-forward dev to the release commit, or replay its metadata on top of dev.

    Returns True when dev was updated and False when dev already carries the
    same release metadata. Every other git failure raises RuntimeError, so a
    stalled sync can never be reported as an already-synced dev.
    """
    for _ in range(attempts):
        git(repo, "fetch", "origin", "dev")
        dev_tip = git(repo, "rev-parse", "refs/remotes/origin/dev").stdout.strip()
        if is_ancestor(repo, release_commit, dev_tip):
            return False

        # Prefer a fast-forward when the release commit extends dev's history,
        # avoiding a duplicate release commit on dev.
        if is_ancestor(repo, dev_tip, release_commit):
            pushed = git(repo, "push", "origin", f"{release_commit}:refs/heads/dev", check=False)
            if pushed.returncode == 0:
                return True
            if not push_lost_race(pushed):
                raise RuntimeError(pushed.stderr.strip() or pushed.stdout.strip())
            continue

        # dev already moved past the release, so replay only the release
        # metadata: merging main would drag unrelated main-only commits onto dev.
        git(repo, "checkout", "--detach", dev_tip)
        git(repo, "checkout", release_commit, "--", *RELEASE_PATHS)
        if git(repo, "diff", "--cached", "--quiet", check=False).returncode == 0:
            return False
        message = release_subject(repo, release_commit)
        git(repo, *BOT_IDENTITY, "commit", "-m", message)

        pushed = git(repo, "push", "origin", "HEAD:refs/heads/dev", check=False)
        if pushed.returncode == 0:
            return True

        git(repo, "reset", "--hard", dev_tip)
        if not push_lost_race(pushed):
            raise RuntimeError(pushed.stderr.strip() or pushed.stdout.strip())

    raise RuntimeError("dev kept advancing while release metadata was being synchronized")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("release_commit", help="main-branch release commit SHA to sync")
    parser.add_argument("--repo", default=".", help="checked-out git repository")
    args = parser.parse_args()
    try:
        changed = sync_release_commit(args.repo, args.release_commit)
    except RuntimeError as error:
        print(f"Release metadata sync failed: {error}", file=sys.stderr)
        raise SystemExit(1) from error
    print("Release metadata pushed to dev" if changed else "dev already contains this release metadata")


if __name__ == "__main__":
    main()
