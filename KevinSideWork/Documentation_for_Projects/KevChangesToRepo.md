# KevRepoChangesWithGuidance.md

*A running log of the exact changes we made to get traces flowing from the **ADK** (Python) and the **GenMedia MCPs** (Go), plus how to verify, common pitfalls, and how to roll back.*

---

## TL;DR

* **Problem:** We were generating media fine, but **no traces** (or intermittent traces). Logs showed gRPC exporter errors like `name resolver error: produced zero addresses`, and our `.env` had **broken multi‑line OTEL values**.
* **Root causes:**

  1. **Default gRPC export** + HTTPS endpoint → connection errors.
  2. **OTEL\_* vars in `.env`*\* overriding our code to gRPC again.
  3. A **newline in the OTLP headers** (`OTEL_EXPORTER_OTLP_TRACES_HEADERS`) broke header parsing.
* **Fix (Python/ADK):**

  * **Remove every `OTEL_*` line from `genmedia_agent/.env`** so the app doesn’t force gRPC.
  * Add a **`sitecustomize.py`** inside the ADK venv to load `.env`, **build headers from `ARIZE_*`**, and **force HTTP/protobuf** endpoints.
* **Fix (Go MCPs):**

  * Replace each MCP’s `otel.go` with a **HTTP‑first exporter** (`otlptracehttp`) that **falls back** to gRPC only if explicitly requested.
  * Add/align OTEL deps, **rebuild all MCP binaries** into `~/go/bin`.
* **Verify:**

  * Quick STDIO ping: `echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | ~/go/bin/mcp-imagen-go`.
  * Run ADK, generate an image, confirm traces under projects/services like **`genmedia-adk`** and **`mcp-imagen-go`**.
* **Critical gotcha:** The headers export must be **a single line**: `space_id=...,api_key=...`. No wrapping.

---

## Repo & Paths We Touched

* **ADK app (Python):**

  * `~/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/sample-agents/adk`
  * ADK’s venv site-packages (Python 3.13 on this machine):

    * `.../adk/.venv/lib/python3.13/site-packages/sitecustomize.py`
  * ADK env file we load:

    * `.../adk/genmedia_agent/.env`
* **Go MCP workspace:**

  * `~/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/mcp-genmedia-go`
  * Modules:

    * `mcp-avtool-go`, `mcp-chirp3-go`, `mcp-gemini-go`, `mcp-imagen-go`, `mcp-lyria-go`, `mcp-veo-go`
  * Binaries end up in: `$(go env GOBIN)` → typically `~/go/bin`

---

## What Was Failing (Symptoms)

* Intermittent/no traces in Arize.
* Go logs periodically: `traces export: exporter export timeout: rpc error: code = Unavailable desc = name resolver error: produced zero addresses`.
* `.env` contained line breaks/spacing issues on OTEL values, and those were re‑forcing **gRPC** even after we tried to switch to HTTP.

---

## Design of the Fix

1. **Single source of truth for credentials** using `ARIZE_SPACE_ID` + `ARIZE_API_KEY`.
2. **HTTP/protobuf** everywhere by default for OTLP, which:

   * Works reliably for both Python SDKs and Go MCPs when pointing at `https://otlp.arize.com`.
   * Avoids gRPC DNS/TLS footguns.
3. **No OTEL\_* in `.env`*\* inside the app repo to prevent accidental overrides.
4. Programmatic **header construction** from `ARIZE_*` if headers aren’t set.

---

## Step‑by‑Step Changes (with commands)

### A) ADK (Python) changes

**Directory:** `~/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/sample-agents/adk`

1. **Back up and clean the app `.env`**

   * Remove *all* `OTEL_*` lines so the app doesn’t force gRPC again.

   ```bash
   cd ~/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/sample-agents/adk
   cp genmedia_agent/.env genmedia_agent/.env.bak.$(date +%s) 2>/dev/null || true
   # keep everything except OTEL_* lines
   grep -v '^OTEL_' genmedia_agent/.env > genmedia_agent/.env.clean || true
   mv genmedia_agent/.env.clean genmedia_agent/.env
   ```

2. **Install a safe `sitecustomize.py` into the venv**

   * This runs on **every Python startup** in this venv and:

     * loads `.env` (root + `genmedia_agent/.env`),
     * builds OTLP headers from `ARIZE_*` if needed,
     * **forces HTTP/protobuf** endpoints.

   ```bash
   source .venv/bin/activate
   python - <<'PY'
   ```

import site, pathlib, textwrap, shutil, os
pkg = pathlib.Path(site.getsitepackages()\[0])
sc  = pkg / "sitecustomize.py"
if sc.exists(): shutil.copy2(sc, sc.with\_suffix(".py.bak"))
sc.write\_text(textwrap.dedent("""
import os

# Best-effort load of .env files

try:
from dotenv import load\_dotenv
load\_dotenv()  # ./.env if present
ga = os.path.join(os.getcwd(), "genmedia\_agent", ".env")
if os.path.isfile(ga):
load\_dotenv(ga, override=False)
except Exception:
pass

# Force HTTP OTLP defaults

os.environ.setdefault("OTEL\_TRACES\_EXPORTER", "otlp")
os.environ.setdefault("OTEL\_EXPORTER\_OTLP\_PROTOCOL", "http/protobuf")
os.environ.setdefault("OTEL\_EXPORTER\_OTLP\_TRACES\_PROTOCOL", "http/protobuf")
os.environ.setdefault("OTEL\_EXPORTER\_OTLP\_ENDPOINT", "[https://otlp.arize.com](https://otlp.arize.com)")
os.environ.setdefault("OTEL\_EXPORTER\_OTLP\_TRACES\_ENDPOINT", "[https://otlp.arize.com/v1/traces](https://otlp.arize.com/v1/traces)")

# Build headers from ARIZE\_\* if not already set

if not os.getenv("OTEL\_EXPORTER\_OTLP\_TRACES\_HEADERS"):
sid = os.getenv("ARIZE\_SPACE\_ID", "").strip()
key = os.getenv("ARIZE\_API\_KEY", "").strip()
if sid and key:
hdr = f"space\_id={sid},api\_key={key}"
os.environ.setdefault("OTEL\_EXPORTER\_OTLP\_TRACES\_HEADERS", hdr)
os.environ.setdefault("OTEL\_EXPORTER\_OTLP\_HEADERS", hdr)
"""))
print(f"✅ wrote {sc}")
PY

````

3) **Runtime exports (shell) to double‑ensure HTTP** *(optional but useful when experimenting)*
```bash
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
export OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=http/protobuf
export OTEL_EXPORTER_OTLP_ENDPOINT="https://otlp.arize.com"
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="https://otlp.arize.com/v1/traces"
export OTEL_EXPORTER_OTLP_HEADERS="space_id=$ARIZE_SPACE_ID,api_key=$ARIZE_API_KEY"
export OTEL_EXPORTER_OTLP_TRACES_HEADERS="$OTEL_EXPORTER_OTLP_HEADERS"
# Optional: sample 100%
export OTEL_TRACES_SAMPLER=always_on
export OTEL_TRACES_SAMPLER_ARG=1.0
````

4. **Run the ADK and generate something** (e.g., Imagen) to produce spans.

### B) MCPs (Go) changes

**Workspace:** `~/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/mcp-genmedia-go`

1. **Replace each server’s `otel.go` with our HTTP‑first version**

   * We wrote a common `otel.go` that:

     * Reads OTEL envs; if headers missing, falls back to `ARIZE_SPACE_ID`/`ARIZE_API_KEY`.
     * Prefers **`otlptracehttp`** with TLS → endpoint `https://otlp.arize.com/v1/traces`.
     * Only falls back to gRPC if explicitly requested.
   * We dropped this file into: `mcp-avtool-go/otel.go`, `mcp-chirp3-go/otel.go`, `mcp-gemini-go/otel.go`, `mcp-imagen-go/otel.go`, `mcp-lyria-go/otel.go`, `mcp-veo-go/otel.go`.

2. **Ensure the new exporter packages are in each module**

   ```bash
   cd ~/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/mcp-genmedia-go
   for mod in mcp-avtool-go mcp-chirp3-go mcp-gemini-go mcp-imagen-go mcp-lyria-go mcp-veo-go; do
     (
       cd "$mod"
       go get \
         go.opentelemetry.io/otel/exporters/otlp/otlptrace@v1.36.0 \
         go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp@v1.36.0 \
         go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.36.0
       go mod tidy
     )
   done
   ```

3. **Sync workspace and install all binaries**

   ```bash
   go work sync
   go install ./mcp-avtool-go ./mcp-chirp3-go ./mcp-gemini-go ./mcp-imagen-go ./mcp-lyria-go ./mcp-veo-go
   ```

   Binaries should now be in:

   ```bash
   BIN_DIR="$(go env GOBIN)"; [ -z "$BIN_DIR" ] && BIN_DIR="$(go env GOPATH)/bin"
   ls "$BIN_DIR"/mcp-*
   # expect: mcp-avtool-go mcp-chirp3-go mcp-gemini-go mcp-imagen-go mcp-lyria-go mcp-veo-go
   ```

4. **Quick STDIO ping (also exercises tracer init)**

   ```bash
   echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | "$BIN_DIR/mcp-imagen-go" | head -n 20
   ```

5. **(Optional) Force HTTP in the shell for local MCP-only tests**

   ```bash
   export OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=http/protobuf
   export OTEL_EXPORTER_OTLP_ENDPOINT="https://otlp.arize.com"
   export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="https://otlp.arize.com/v1/traces"
   export OTEL_EXPORTER_OTLP_HEADERS="space_id=$ARIZE_SPACE_ID,api_key=$ARIZE_API_KEY"
   export OTEL_EXPORTER_OTLP_TRACES_HEADERS="$OTEL_EXPORTER_OTLP_HEADERS"
   ```

---

## How We Verified

1. **Go MCP sanity:** `tools/list` returns the tool schemas and logs show tracer initialized; no gRPC resolver errors.
2. **ADK end‑to‑end:** start the web server, request an image (e.g., *“River Rapids”*), see logs like *“Generated 1 image … saved to GCS …”* and confirm a new trace appears under the Arize project.
3. **Arize UI:** look for services named like `genmedia-adk` and individual MCPs such as `mcp-imagen-go`.

---

## Gotchas & Pitfalls We Hit (so you don’t)

* **Newline in headers:**

  * Don’t split this across lines — it must be **one line**:

    ```bash
    export OTEL_EXPORTER_OTLP_TRACES_HEADERS="space_id=$ARIZE_SPACE_ID,api_key=$ARIZE_API_KEY"
    ```
* **`.env` accidentally forcing gRPC:**

  * Any `OTEL_*` in `genmedia_agent/.env` can override our HTTP settings.
  * Keep `.env` to app creds (`ARIZE_*`, `PROJECT_ID`, etc.), not transport knobs.
* **gRPC “produced zero addresses”:**

  * Classic sign the exporter tried gRPC against an HTTPS URL or had TLS/DNS issues. Using HTTP/protobuf avoids this class entirely.
* **Multiple places load env:**

  * We load both root `.env` and `genmedia_agent/.env`. Ensure values are consistent. If in doubt, **unset** `OTEL_*` in the shell before launching.
* **Sampling:**

  * If traces seem sparse, set `OTEL_TRACES_SAMPLER=always_on` and `OTEL_TRACES_SAMPLER_ARG=1.0` while validating.

---

## Rollback Plan

* **ADK (Python):**

  * Restore `sitecustomize.py` backup:

    ```bash
    cp -v .venv/lib/python3.13/site-packages/sitecustomize.py.bak \
          .venv/lib/python3.13/site-packages/sitecustomize.py
    ```
  * Restore `.env` backup:

    ```bash
    mv -v genmedia_agent/.env.bak.* genmedia_agent/.env  # pick the right timestamp
    ```
* **MCPs (Go):**

  * Each module has `otel.go.bak.<timestamp>` (if you saved one). Restore and rebuild:

    ```bash
    for mod in mcp-avtool-go mcp-chirp3-go mcp-gemini-go mcp-imagen-go mcp-lyria-go mcp-veo-go; do
      [ -f "$mod/otel.go.bak" ] && mv -v "$mod/otel.go.bak" "$mod/otel.go"
    done
    go work sync && go install ./...
    ```

---

## Repeatable Checklists

### Launch & Verify (ADK)

* [ ] `genmedia_agent/.env` contains **no** `OTEL_*` lines.
* [ ] `ARIZE_SPACE_ID` and `ARIZE_API_KEY` are present.
* [ ] `sitecustomize.py` present in the ADK venv.
* [ ] (Optional) Shell exports force HTTP/protobuf (see above).
* [ ] Start ADK, generate an image → trace visible in Arize.

### Build & Verify (MCPs)

* [ ] `otel.go` in each MCP uses HTTP-first exporter.
* [ ] `go work sync` succeeds.
* [ ] `go install ./mcp-...` places binaries in `~/go/bin`.
* [ ] `tools/list` STDIO ping works and logs show tracer init.

---

## Appendix A — Final `sitecustomize.py` (Python / ADK)

```python
import os
# Best-effort load of .env files
try:
    from dotenv import load_dotenv
    load_dotenv()  # ./.env if present
    ga = os.path.join(os.getcwd(), "genmedia_agent", ".env")
    if os.path.isfile(ga):
        load_dotenv(ga, override=False)
except Exception:
    pass

# Force HTTP OTLP defaults
os.environ.setdefault("OTEL_TRACES_EXPORTER", "otlp")
os.environ.setdefault("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
os.environ.setdefault("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "http/protobuf")
os.environ.setdefault("OTEL_EXPORTER_OTLP_ENDPOINT", "https://otlp.arize.com")
os.environ.setdefault("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "https://otlp.arize.com/v1/traces")

# Build headers from ARIZE_* if not already set
if not os.getenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS"):
    sid = os.getenv("ARIZE_SPACE_ID", "").strip()
    key = os.getenv("ARIZE_API_KEY", "").strip()
    if sid and key:
        hdr = f"space_id={sid},api_key={key}"
        os.environ.setdefault("OTEL_EXPORTER_OTLP_TRACES_HEADERS", hdr)
        os.environ.setdefault("OTEL_EXPORTER_OTLP_HEADERS", hdr)
```

## Appendix B — Header exports (single line!)

```bash
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
export OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=http/protobuf
export OTEL_EXPORTER_OTLP_ENDPOINT="https://otlp.arize.com"
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="https://otlp.arize.com/v1/traces"
export OTEL_EXPORTER_OTLP_HEADERS="space_id=$ARIZE_SPACE_ID,api_key=$ARIZE_API_KEY"
export OTEL_EXPORTER_OTLP_TRACES_HEADERS="$OTEL_EXPORTER_OTLP_HEADERS"
```

## Appendix C — One‑shot MCP sanity ping

```bash
BIN_DIR="$(go env GOBIN)"; [ -z "$BIN_DIR" ] && BIN_DIR="$(go env GOPATH)/bin"
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | "$BIN_DIR/mcp-imagen-go" | head -n 20
```

## Appendix D — Minimal `otel.go` (conceptual outline)

> We placed a shared HTTP‑first `otel.go` into each MCP module. Core behavior:

* Parse `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` or default to `https://otlp.arize.com/v1/traces`.
* Parse headers from `OTEL_*` or build from `ARIZE_*`.
* Create **HTTP exporter** (`otlptracehttp.New`) with TLS; if `OTEL_EXPORTER_OTLP_PROTOCOL=grpc`, use gRPC exporter instead.
* Set resource attrs (`service.name`, `service.version`), and register a `BatchSpanProcessor` on a `TracerProvider`.

*(Exact code lives in each module at `mcp-*/otel.go` in the working tree.)*

---

## Changelog (human notes)

* **\[2025‑09‑03]** Cleaned `.env`, added venv `sitecustomize.py` (HTTP‑first), replaced MCP `otel.go` across modules, added `otlptracehttp` deps, rebuilt binaries, validated traces for ADK + MCP.
