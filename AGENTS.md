# Repository Guidelines

These guidelines help contributors work consistently across the GenMedia Creative Studio codebase.

## Project Structure & Module Organization
- Core app entrypoint: `main.py` (Mesop UI + Flask endpoints).
- Configuration: `config/` and `.env` (see `env_template`).
- App logic and types: `models/`, `prompts/`.
- Experiments and prototypes: `experiments/` (stand‑alone apps, read its README).
- Media assets (git‑ignored or lightweight): `images/`, `videos/`, `music/`, `screenshots/`, `svg_icon/`.
- Docs and guides: `README.md`, `GEMINI.md`, `media-handling-issues-evaluation.md`.

## Build, Test, and Development Commands
- Create venv: `python3 -m venv venv && source venv/bin/activate`.
- Install deps: `pip install -r requirements.txt`.
- Run locally (Mesop): `mesop main.py`.
- Set required env vars before running:
  - `export PROJECT_ID=$(gcloud config get project)`
  - `export IMAGE_CREATION_BUCKET=<your-bucket>`
- Deploy (Cloud Run example): see `README.md` for `gcloud run deploy` with `PROJECT_ID` and `IMAGE_CREATION_BUCKET`.

## Coding Style & Naming Conventions
- Python 3: follow PEP 8; 4‑space indents; prefer type hints and docstrings.
- Names: `snake_case` for files/functions, `PascalCase` for classes, constants `UPPER_SNAKE_CASE`.
- Strings: prefer f‑strings; keep lines readable (~100–120 chars).
- Imports: standard library → third‑party → local, grouped and alphabetized.
- Avoid spaces in new folder/file names; use `snake_case`.

## Testing Guidelines
- No formal test suite yet. For new code, add `pytest` tests under `tests/` using `test_*.py` pattern.
- Example: `pip install pytest && pytest -q`.
- For UI flows, document manual steps in PR description and include screenshots.

## Commit & Pull Request Guidelines
- Use Conventional Commits found in history (e.g., `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`).
- Scope small, focused PRs. Include:
  - Clear description, linked issue(s), and rationale.
  - Screenshots or clips for UI/media changes.
  - Notes on env/config changes and rollback considerations.
- Update relevant docs when behavior or commands change.

## Security & Configuration
- Never commit secrets or service keys. Use `.env` (based on `env_template`) and GCP IAM roles (`aiplatform.user`, `storage.objectUser`).
- Large/generated media should remain outside Git history; keep under the provided media folders and confirm they’re ignored.

