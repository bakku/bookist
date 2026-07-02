---
name: commit
description: >-
  Use when the user says "commit", "stage", "git commit", or asks to commit changes.
  Provides a consistent git commit workflow that matches this project's existing
  commit style — imperative subject lines, structured body with bullet details,
  and appropriate context.
---

# Commit Workflow

When the user asks you to commit, follow these steps:

## 1. Understand the project's commit style

Check the last 5-10 commits to match the tone and structure:

```bash
git log --oneline -10
git log --format="%h %s%+b" -3
```

This project uses:
- **Subject line**: imperative mood, capitalised, no trailing period
- **Body**: blank line after subject, bullet lists or short paragraphs
- **Scope**: naturally scoped by the subject (e.g. "Add language, publisher, ... fields to books")

## 2. Review what changed

```bash
git diff --stat       # file summary
git diff              # full diff for commit message content
```

Scan the diff to compose an accurate subject and body. Do not include
auto-formatting or whitespace-only changes in the description unless they are
the point of the commit.

## 3. Stage

```bash
git add -A
```

Stage all changes unless the user specifies a subset.

## 4. Commit

Write a message with a short subject line and a body that explains *what* and
*why*, not a line-by-line listing of every file touched.

```bash
git commit -m "Subject line in imperative mood

Body paragraph explaining the change and the motivation.

- Bullet list of concrete additions, if helpful
- Focus on user-visible or API-visible changes"
```

## Example from this project

```
Add language, publisher, edition, format, purchased_at, pages, notes,
and published date fields to books

Extend the book model with 10 new columns to support richer library
cataloging:

- language, publisher, edition, notes: nullable text
- format: nullable enum (hardback, paperback, epub)
- purchased_at: nullable ISO date text
- pages: nullable integer
- published_year, published_month, published_day: nullable integers

Add Format type with typed consts and validation in the service layer.
Wire new fields through the repository, HTTP API, and CLI.
Add comprehensive tests covering all fields at every layer.
```

## After committing

Optionally offer to push if the user wants.
