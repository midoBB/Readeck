# SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
#
# SPDX-License-Identifier: AGPL-3.0-only

import io
import re
from argparse import ArgumentParser
from pathlib import Path

from babel.messages.catalog import Catalog
from babel.messages.extract import extract_from_file
from babel.messages.mofile import write_mo
from babel.messages.pofile import read_po, write_po

HERE = Path(__file__).parent
ROOT = (HERE / "..").resolve()

CATALOG_HEADER = """\
# Translations template for PROJECT.
# SPDX-FileCopyrightText: © YEAR Readeck <translate@readeck.com>
#
# SPDX-License-Identifier: AGPL-3.0-only
#"""

CATALOG_OPTIONS = {
    "header_comment": CATALOG_HEADER,
    "project": "Readeck",
    "version": "1.0.0",
    "copyright_holder": "Readeck",
    "msgid_bugs_address": "translate@readeck.com",
    "last_translator": "Readeck <translate@readeck.com>",
    "language_team": "Readeck <translate@readeck.com>",
}


def tok(name: str):
    def s(scanner, token):
        return (name, token, scanner.match)

    return s


class GoScanner:
    scanner = re.Scanner(
        [
            (r"//.*$", tok("COMMENT")),
            (r'"(?:[^"\\]|\\.)*"', tok("STRING")),
            (r"`", tok("RAW_DELIM")),
            (r"[0-9]+", tok("INT")),
            (r",", tok("COMMA")),
            (r"\+", tok("PLUS")),
            (r"\(", tok("OPEN")),
            (r"\)", tok("CLOSE")),
            (r"Gettext|Ngettext|Pgettext|Npgettext", tok("FUNC")),
            (r".", None),
        ],
        flags=re.DOTALL | re.I,
    )

    def __init__(self, fp) -> None:
        self.fp = fp
        self.line = "\n"
        self.lineno = 0
        self.tokens = []
        self.idx = -1

        self._next()

    def _nextline(self):
        self.line = self.fp.readline().decode("utf-8")
        self.lineno += 1

        self.tokens, _ = self.scanner.scan(self.line)
        self.idx = -1

    def _next(self):
        # Advance to next token
        if len(self.tokens) > 0 and self.idx + 1 < len(self.tokens):
            self.idx += 1
            return True

        self._nextline()
        while len(self.tokens) == 0 and not self.eof:
            self._nextline()

        if self.eof:
            self.idx = -1
            return None

        self.idx = 0
        return True

    @property
    def eof(self) -> bool:
        return self.line == ""

    @property
    def token(self) -> str:
        return self.tokens[self.idx][0] if self.idx >= 0 else None

    @property
    def value(self) -> str:
        return self.tokens[self.idx][1] if self.idx >= 0 else None

    @property
    def match(self):
        return self.tokens[self.idx][2] if self.idx >= 0 else None

    def extract(self):
        while not self.eof:
            if self.token == "FUNC":
                res = self.visit_function()
                if res is not None:
                    yield res
            self._next()

    def extract_strings(self):
        for lineno, func, args in self.extract():
            if len(args) == 0:
                continue
            if all(x is None for x in args):
                continue

            messages = []
            for m in args:
                if m is None:
                    break
                messages.append(m)
            yield (lineno, func, messages, [])

    def visit_function(self):
        name = self.value
        lineno = self.lineno
        self._next()

        args = []
        s = None
        while self.token:
            if self.token in ("STRING", "RAW_DELIM"):
                s = self.visit_string()
            if self.token in ("COMMA", "CLOSE"):
                args.append(s)
                s = None

            if self.token == "CLOSE":
                break

            self._next()

        return (lineno, name, args)

    def visit_string(self):
        res = []
        while self.token in ("STRING", "RAW_DELIM", "PLUS"):
            if self.token == "RAW_DELIM":
                res.append(self.visit_raw_string())
            if self.token == "STRING":
                res.append(self.value[1:-1].replace('\\"', '"'))
            elif self.token == "LITERAL":
                res.append(self.value[1:-1])

            self._next()

        return "".join(res)

    def visit_raw_string(self):
        res = []
        start = self.match.span()[1]
        while True:
            next = [
                i
                for i, x in enumerate(self.tokens[self.idx + 1 :])
                if x[0] == "RAW_DELIM"
            ]
            if len(next) > 0:
                self.idx = self.idx + next[0] + 1
                end = self.match.span()[0]
                res.append(self.line[start:end])
                break

            res.append(self.line[start:])
            start = 0
            self._nextline()

        return "".join(res)


class JetScanner(GoScanner):
    scanner = re.Scanner(
        [
            (r'"(?:[^"\\]|\\.)*"', tok("STRING")),
            (r"`", tok("RAW_DELIM")),
            (r"[0-9]+", tok("INT")),
            (r",", tok("COMMA")),
            (r"\+", tok("PLUS")),
            (r"\(", tok("OPEN")),
            (r"\)", tok("CLOSE")),
            (r"gettext|ngettext|pgettext|npgettext", tok("FUNC")),
            (r".", None),
        ],
        flags=re.DOTALL,
    )

    def extract(self):
        while not self.eof:
            if self.token in ("STRING", "RAW_DELIM"):
                # We can have function calls in attributes (double quoted strings)
                # The easy/dirty path is to parse again the string content.
                s = (
                    self.visit_string()
                    if self.token == "STRING"
                    else self.visit_raw_string()
                )

                sub = JetScanner(io.BytesIO(s.encode()))
                for x in sub.extract():
                    yield x
            if self.token == "FUNC":
                res = self.visit_function()
                if res is not None:
                    yield res
            self._next()


class TmplScanner(JetScanner):
    scanner = re.Scanner(
        [
            (r'"(?:[^"\\]|\\.)*"', tok("STRING")),
            (r"`", tok("RAW_DELIM")),
            (r"[0-9]+", tok("INT")),
            (r",", tok("COMMA")),
            (r"\ ", tok("SPACE")),
            (r"\+", tok("PLUS")),
            (r"\(", tok("OPEN")),
            (r"\}\}", tok("CLOSE")),
            (r"Gettext|Ngettext|Pgettext|Npgettext", tok("FUNC")),
            (r".", None),
        ],
        flags=re.DOTALL,
    )

    def visit_function(self):
        name = self.value
        lineno = self.lineno
        self._next()
        self._next()

        args = []
        s = None
        while self.token:
            if self.token in ("STRING", "RAW_DELIM"):
                s = self.visit_string()
            if self.token in ("SPACE", "CLOSE"):
                args.append(s)
                s = None

            if self.token == "CLOSE":
                break

            self._next()

        return (lineno, name, args)


def extract_go(fileobj, keywords, comment_tags, options):
    return GoScanner(fileobj).extract_strings()


def extract_jet(fileobj, keywords, comment_tags, options):
    return JetScanner(fileobj).extract_strings()


def extract_tmpl(fileobj, keywords, comment_tags, options):
    return TmplScanner(fileobj).extract_strings()


METHODS = {
    ".go": extract_go,
    ".jet.html": extract_jet,
    ".tmpl": extract_tmpl,
}

KEYWORDS = {
    "Gettext": None,
    "Ngettext": (1, 2),
    "Pgettext": ((1, "c"), 2),
    "Npgettext": ((1, "c"), 2, 3),
    "gettext": None,
    "ngettext": (1, 2),
    "pgettext": ((1, "c"), 2),
    "npgettext": ((1, "c"), 2, 3),
}


def extract(_):
    template = Catalog(**CATALOG_OPTIONS)

    for f in ROOT.rglob("*"):
        method = METHODS.get("".join(f.suffixes))
        if method is None:
            continue
        for lineno, message, comments, context in extract_from_file(
            method, f, keywords=KEYWORDS
        ):
            template.add(
                message,
                None,
                [(str(f.relative_to(ROOT)), lineno)],
                auto_comments=comments,
                context=context,
            )

    translations = HERE / "translations"
    dest = translations / "messages.pot"
    with dest.open("wb") as fp:
        write_po(fp, template, sort_by_file=True, include_lineno=True)
        print(f"{dest} written")


def update(_):
    translations = HERE / "translations"
    with (translations / "messages.pot").open("rb") as fp:
        template = read_po(fp)

    dirs = [x for x in translations.iterdir() if x.is_dir()]
    for p in dirs:
        po_file = p / "messages.po"
        if po_file.exists():
            with po_file.open("rb") as fp:
                catalog = read_po(fp, locale=p.name, domain=po_file.name)
        else:
            catalog = Catalog(
                **CATALOG_OPTIONS,
                locale=p.name,
                domain=po_file.name,
            )

        catalog.update(template)

        with po_file.open("wb") as fp:
            write_po(fp, catalog, sort_by_file=True, include_lineno=False)
            print(f"{po_file} written")


def compile(_):
    translations = HERE / "translations"
    mo_files = translations.glob("*/messages.po")

    for filename in mo_files:
        with open(filename, "rb") as fp:
            catalog = read_po(fp)

        dest = filename.with_suffix(".mo")
        with open(dest, "wb") as fp:
            write_mo(fp, catalog)
            print(f"{dest} written")


def main():
    parser = ArgumentParser()
    subparsers = parser.add_subparsers(required=True)

    p_extract = subparsers.add_parser("extract", help="Extract messages")
    p_extract.set_defaults(func=extract)

    p_update = subparsers.add_parser("update", help="Update strings")
    p_update.set_defaults(func=update)

    p_compile = subparsers.add_parser("compile", help="Compile gettext .mo files")
    p_compile.set_defaults(func=compile)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
