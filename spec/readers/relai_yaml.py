#!/usr/bin/env python3
"""A second reader of relai.yaml (format v0), written from spec/relai-yaml.md alone.

Usage: relai_yaml.py <file>     prints the manifest as JSON.
Standard library only. It exists to show the page is enough to write a reader.
"""
import json
import sys

LISTS = ("provides", "consumes", "links")


def strip_comment(s):
    quote = ""
    for i, c in enumerate(s):
        if quote:
            if c == quote:
                quote = ""
        elif c in "'\"":
            quote = c
        elif c == "#":
            return s[:i].strip()
    return s.strip()


def unquote(s):
    s = s.strip()
    if len(s) >= 2 and s[0] in "'\"" and s[-1] == s[0]:
        return s[1:-1]
    return s


def split_top(s):
    """Split on the commas that are not inside braces, brackets or quotes."""
    out, depth, start, quote = [], 0, 0, ""
    for i, c in enumerate(s):
        if quote:
            if c == quote:
                quote = ""
        elif c in "'\"":
            quote = c
        elif c in "{[":
            depth += 1
        elif c in "}]":
            depth -= 1
        elif c == "," and depth == 0:
            out.append(s[start:i])
            start = i + 1
    return [p.strip() for p in out + [s[start:]] if p.strip()]


def capability_of(mapping):
    """The value of capability: in 'k: v, k: v', or None."""
    for pair in split_top(mapping):
        key, sep, value = pair.partition(":")
        if sep and key.strip() == "capability":
            return unquote(strip_comment(value))
    return None


def inline_list(value):
    names = []
    for part in split_top(value[1:-1]):
        if part.startswith("{") and part.endswith("}"):
            name = capability_of(part[1:-1])
        else:
            name = unquote(part)
        if name:
            names.append(name)
    return names


def block_list(lines, i):
    """Read the '- item' lines after lines[i]; return (names, index of the last line read)."""
    names, open_entry = [], False
    while i + 1 < len(lines):
        raw = lines[i + 1]
        if raw.strip() == "" or raw.strip().startswith("#"):
            i += 1
            continue
        if raw[0] not in " \t-":
            break  # next top-level key
        text = strip_comment(raw)
        if text.startswith("-"):
            item = text[1:].strip()
            key, sep, value = item.partition(":")
            if not item:
                open_entry = True
            elif sep and item[0] not in "'\"":
                open_entry = key.strip() != "capability"
                if not open_entry:
                    names.append(unquote(value))
            else:
                names.append(unquote(item))
                open_entry = False
        elif open_entry:  # a continuation line of the current entry
            name = capability_of(text)
            if name is not None:
                names.append(name)
                open_entry = False
        i += 1
    return names, i


def parse(text):
    manifest = {}
    lines = text.lstrip("﻿").replace("\r\n", "\n").split("\n")
    i = 0
    while i < len(lines):
        line = lines[i]
        if line and line[0] not in " \t#-" and ":" in line:
            key, _, value = line.partition(":")
            key, value = key.strip(), strip_comment(value)
            if key in ("apiVersion", "id", "summary", "conventions", "tasks"):
                manifest[key] = unquote(value)
            elif key == "owners" or key in LISTS:
                if value.startswith("[") and value.endswith("]"):
                    names = inline_list(value)
                elif value in ("", "~", "null"):
                    names, i = block_list(lines, i)
                else:
                    names = []
                manifest[key] = manifest.get(key, []) + names
        i += 1
    return manifest


if __name__ == "__main__":
    with open(sys.argv[1], encoding="utf-8") as f:
        json.dump(parse(f.read()), sys.stdout, indent=2, sort_keys=True)
        print()
