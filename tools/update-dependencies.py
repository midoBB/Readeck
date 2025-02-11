#!/usr/bin/python3

# SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
#
# SPDX-License-Identifier: AGPL-3.0-only

# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "httpx",
# ]
# ///

import os
from contextlib import contextmanager
from datetime import date
from subprocess import call, check_call
from tempfile import TemporaryDirectory

import httpx

API_URL = os.environ["GITHUB_API_URL"]
REPOSITORY = os.environ["GITHUB_REPOSITORY"]
GIT_REPO_USER = os.environ["API_USER"]
GIT_REPO_PASSWORD = os.environ["API_TOKEN"]
GIT_URL = f"https://{GIT_REPO_USER}:{GIT_REPO_PASSWORD}@codeberg.org/{REPOSITORY}.git/"

SITE_CONFIG_REPO = "https://github.com/j0k3r/graby-site-config.git"


@contextmanager
def branch(name: str):
    check_call(["git", "checkout", "-B", name])
    yield name
    check_call(["git", "checkout", "main"])


def commit_changes(files: list[str], message: str):
    check_call(["git", "add"] + files)
    rc = call(["git", "diff-index", "--quiet", "HEAD", "--"] + files)
    if rc == 0:
        return

    # fmt:off
    check_call(
        [
            "git", "commit",
            "-m", message,
            "--no-signoff", "--no-gpg-sign",
        ], env={
            **os.environ,
            "GIT_AUTHOR_NAME": "Readeck Bot",
            "GIT_AUTHOR_EMAIL": "bot@readeck.com",
            "GIT_COMMITTER_NAME": "Readeck Bot",
            "GIT_COMMITTER_EMAIL": "bot@readeck.com",
        },
    )
    # fmt:on


def push_changes(branch_name: str):
    rc = call(["git", "diff-index", "--quiet", "main"])
    if rc == 0:
        return

    check_call(["git", "push", "--force", GIT_URL, branch_name])


def create_pr(branch_name: str):
    rsp = httpx.post(
        f"{API_URL}/repos/{REPOSITORY}/pulls",
        headers={
            "Authorization": f"token {GIT_REPO_PASSWORD}",
        },
        json={
            "base": "main",
            "head": branch_name,
            "title": f"Dependencies update [{date.today()}]",
        },
    )
    if rsp.status_code == 409:
        return
    rsp.raise_for_status()


def update_go_dependencies():
    check_call(["go", "get", "-t", "-u", "-v"])
    check_call(["go", "mod", "tidy"])


def update_js_dependencies():
    # fmt:off
    check_call(
        [
            "npm", "exec", "-y", "--",
            "npm-check-updates",
            "--cwd", "web",
            "-t", "minor",
            "--peer", "--upgrade",
            "--install", "always",
        ]
    )
    # fmt: on
    check_call(["npx", "-y", "update-browserslist-db@latest"])


def update_site_config_files():
    with TemporaryDirectory() as folder:
        # fmt: off
        check_call(
            [
                "git", "clone", "--depth", "1",
                SITE_CONFIG_REPO, "--single-branch",
                "--branch", "master",
                folder,
            ]
        )
        check_call(
            [
                "go", "run", "./tools/ftr",
                folder,
                "pkg/extract/contentscripts/assets/site-config",
            ], env={
                **os.environ,
                "GOWORK": "off",
            },
        )
        # fmt: on


def main():
    with branch("chore/updates") as branch_name:
        update_go_dependencies()
        commit_changes(
            ["go.mod", "go.sum"],
            "Updated Go dependencies",
        )

        update_js_dependencies()
        commit_changes(
            ["web/package.json", "web/package-lock.json"],
            "Updated JS dependencies",
        )

        update_site_config_files()
        commit_changes(
            ["pkg/extract/contentscripts/assets/site-config"],
            "Updated Site Config files",
        )

        rc = call(["git", "diff-index", "--quiet", "main"])
        if rc == 0:
            print("no new updates")
            return

        push_changes(branch_name)
        create_pr(branch_name)


if __name__ == "__main__":
    main()
