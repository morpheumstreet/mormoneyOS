**Yes, Ollama provides a clean way to see which model(s) are currently loaded / "employed" (i.e. resident in memory and ready for inference) via its REST API.**

Use this endpoint:

```
GET http://localhost:11434/api/ps
```

Example with curl:

```bash
curl http://localhost:11434/api/ps
```

Typical response looks like:

```json
{
  "models": [
    {
      "name": "llama3.2:latest",
      "model": "llama3.2:latest",
      "size": 3825819519,
      "digest": "somehashvalue",
      "details": {
        "format": "gguf",
        "family": "llama",
        "families": null,
        "parameter_size": "3B",
        "quantization_level": "Q4_K_M"
      },
      "expires_at": "2026-03-16T03:15:22.123456Z",
      "size_vram": 3825819519,
      "context_length": 131072
    }
  ]
}
```

- If multiple models are loaded at the same time → you'll see multiple entries in the array
- If nothing is loaded → you'll get `"models": []`

This is the standard / official way to check the **currently loaded ("employed") model(s)**.

### Quick comparison of related endpoints

| Endpoint          | Method | What it shows                              | Use when you want to know...                  |
|-------------------|--------|--------------------------------------------|-----------------------------------------------|
| `/api/ps`         | GET    | Models currently loaded in memory          | ← **this is what you asked for**              |
| `/api/tags`       | GET    | All models you have downloaded locally     | What models are available to load             |
| `/api/show`       | POST   | Detailed info about one specific model     | Modelfile / template / parameters of a model  |

So in short:

- Want to know **which model is currently active/loaded**? → `GET /api/ps`
- Want to know what you **can** load? → `GET /api/tags`

That's the idiomatic way using the Ollama API.

**Yes — Ollama has a dedicated API endpoint exactly for showing the currently active / loaded model(s).**

The endpoint is:

```
GET http://localhost:11434/api/ps
```

This returns the models that are **currently loaded into memory** (i.e. "active" and ready for fast inference without reloading).

### Quick examples

**Using curl:**
```bash
curl http://localhost:11434/api/ps
```

**Typical response when one model is active:**
```json
{
  "models": [
    {
      "name": "llama3.2:latest",
      "model": "llama3.2:latest",
      "size": 3825819519,
      "digest": "a2af6cc3eb7fa8be8504abaf9b04e88f17a119ec3f04a3addf55f92841195f5a",
      "details": {
        "format": "gguf",
        "family": "llama",
        "families": null,
        "parameter_size": "3B",
        "quantization_level": "Q4_K_M"
      },
      "expires_at": "2026-03-17T02:45:00Z",
      "size_vram": 3825819519,
      "context_length": 131072
    }
  ]
}
```

- `"models": []` → nothing is currently loaded / active
- Multiple entries → several models are loaded at once (possible since Ollama supports this)
- The `"expires_at"` field shows when the model will auto-unload if no requests come in (controlled by `keep_alive`)

### CLI equivalent (for quick checking without API)
```bash
ollama ps
```

This gives a nice human-readable table:
```
NAME              ID              SIZE      PROCESSOR    UNTIL
llama3.2:latest   abc123...       3.8 GB    100% GPU     5 minutes from now
```

### Quick comparison: related but different endpoints

| What you want                        | Endpoint / Command          | Returns                              |
|--------------------------------------|-----------------------------|--------------------------------------|
| Currently active / loaded model(s)   | `GET /api/ps`  or `ollama ps` | Models in memory right now           |
| All models you have downloaded       | `GET /api/tags` or `ollama list` | Available models (not necessarily loaded) |
| Details about one specific model     | `POST /api/show`            | Modelfile, template, params, etc.    |

So if your question is literally **"show the current active model"**, then `/api/ps` is the correct and official way.

(As of 2026 this is still the standard method — no major changes to this part of the API.)
