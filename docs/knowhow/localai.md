**Yes, LocalAI (the project at localai.io) has a way to check the currently loaded / active model(s) via its API.**

LocalAI runs an **OpenAI-compatible API** by default (at `http://localhost:8080/v1/`), but it also provides a few **native endpoints** for system/model status that go beyond pure OpenAI compatibility.

### Best / most direct way: native LocalAI endpoint for loaded models

Use this:

```
GET http://localhost:8080/system
```

(or `http://localhost:8080/v1/system` — both work in recent versions)

This returns a JSON object including:

- Available backends
- **Currently loaded models** (exactly what you're asking for — the active ones in memory)
- Version info

Example response snippet (simplified):

```json
{
  "backends": ["llama.cpp", "diffusers", "whisper", ...],
  "loaded_models": [
    {
      "id": "phi-3-mini-4k-instruct",
      "backend": "llama.cpp"
    },
    {
      "id": "my-custom-model"
    }
  ],
  "version": "v2.x.x"
}
```

- If `"loaded_models": []` → no model is currently active/loaded
- Multiple entries → several models are loaded simultaneously (LocalAI supports this, though it usually loads one at a time unless configured otherwise)

This is the closest equivalent to Ollama's `GET /api/ps`.

### OpenAI-compatible way (partial / indirect)

You can also query:

```
GET http://localhost:8080/v1/models
```

(or `http://localhost:8080/models`)

This is the standard OpenAI `/v1/models` endpoint and lists **all models that LocalAI can serve** (i.e., all models present in your models directory that have valid config files).

- It does **not** strictly show only what's loaded in memory right now
- In practice, many LocalAI setups load models on-demand (or keep the last-used one loaded), so the "active" one is often the one you see used in recent requests
- If you have `f16` or similar flags to keep models resident, it aligns better

But for precise "currently loaded / active" status → use `/system` instead.

### CLI / quick check (no API needed)

If you're running LocalAI via Docker or binary:

- There isn't a direct `local-ai ps` equivalent like Ollama
- But you can watch logs when a model loads/unloads (very verbose with `--debug`)
- Or just curl the `/system` endpoint as above

### Quick comparison: Ollama vs LocalAI

| Feature                          | Ollama                          | LocalAI                                      |
|----------------------------------|---------------------------------|----------------------------------------------|
| Endpoint for currently loaded    | `GET /api/ps`                   | `GET /system` (or `/v1/system`)              |
| Shows memory-resident models     | Yes, very explicit              | Yes (in `"loaded_models"`)                   |
| OpenAI-compatible `/v1/models`   | `GET /api/tags` (downloaded)    | `GET /v1/models` (all available/configured)  |
| Multi-model loaded at once       | Yes                             | Yes (but less common)                        |
| Default port                     | 11434                           | 8080                                         |

So in short:

- Want the **current active/loaded model(s)** in LocalAI? → `GET http://localhost:8080/system`
- Want to see what you **can** use? → `GET http://localhost:8080/v1/models`

(As of 2026 this is still current — LocalAI's native `/system` endpoint remains the go-to for loaded model status.)