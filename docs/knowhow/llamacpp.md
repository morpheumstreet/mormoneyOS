**Yes, llama.cpp's server (llama-server) supports an API endpoint to view/list the current active/loaded model(s).**

It implements the standard OpenAI-compatible `/v1/models` endpoint, which returns information about the currently loaded model(s). This is the most straightforward and compatible way to query it.

### How to use it
Start the server normally (e.g.):
```bash
./llama-server -m models/llama-3.1-8b-instruct-q5_k_m.gguf --port 8080
# or with alias for nicer model name
./llama-server -m models/llama-3.1-8b-instruct-q5_k_m.gguf --alias "Llama-3.1-8B-Instruct" --port 8080
```

Then query the endpoint:
```bash
curl http://localhost:8080/v1/models
```

Typical response looks something like this (JSON):
```json
{
  "object": "list",
  "data": [
    {
      "id": "Llama-3.1-8B-Instruct",  // or the filename/alias you set
      "object": "model",
      "created": 1730000000,           // approximate timestamp
      "owned_by": "llama.cpp",
      "permission": [...],
      "root": "Llama-3.1-8B-Instruct",
      "parent": null
    }
  ]
}
```

- In single-model mode (the default/classic way), it lists **one model** — the currently loaded one.
- The `id` field usually reflects the `--alias` if you set one, the filename, or a fallback name.
- You can use this with the official OpenAI Python client too:

```python
from openai import OpenAI

client = OpenAI(base_url="http://localhost:8080/v1", api_key="dummy")

models = client.models.list()
print(models.data[0].id)  # e.g. "Llama-3.1-8B-Instruct"
```

### Other ways to check the current model
- **Built-in web UI**: Open `http://localhost:8080` (or your port) in a browser — it shows the loaded model name right at the top or in the settings/chat header.
- **/props** or **/metrics** endpoints: These exist for more server stats (e.g. `curl http://localhost:8080/props`), and sometimes include model-related info like context size, but `/v1/models` is cleaner for just "what model is active".
- **Command-line startup log**: When you launch the server, it prints the model path/filename immediately.

### Notes on multi-model support (as of 2026)
- Classic `llama-server` loads **one model at a time** (you'd restart or use external tools like llama-swap for switching).
- Recent llama.cpp versions added **router mode** (start without `-m`, it auto-discovers models from cache), with a `/models` endpoint (note: no `/v1/`) that lists **all discovered models** + their load status (loaded/loading/unloaded). But for the standard OpenAI-compatible path, stick to `/v1/models` — it shows what's actively serving requests right now.

This keeps llama.cpp's server lightweight while still being useful for tools that expect OpenAI-style model listing (e.g. LangChain, some GUIs, or scripts that validate the backend).

If you're building something automated or integrating with a frontend, `/v1/models` is reliable and widely supported. Let me know if you want examples for specific use cases (e.g. switching models dynamically, checking in a script, or pairing with OpenWebUI)!