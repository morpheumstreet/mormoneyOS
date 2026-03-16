**Yes, LM Studio has a built-in server API** — and it's one of its strongest features for developers.

### Server API Overview
- You enable it from the **Developer** tab (or "Local Server" section) in the LM Studio desktop app.
- Toggle "Start server" (it usually runs on `http://localhost:1234` by default, with the OpenAI-compatible path at `/v1`).
- It supports both:
  - **OpenAI-compatible endpoints** (e.g., `/v1/chat/completions`, `/v1/embeddings`, `/v1/models`, and even newer ones like `/v1/responses` for stateful interactions).
  - **Native LM Studio REST API** (e.g., under `/api/v1/` or older `/api/v0/` paths for model management).
- This makes it drop-in compatible with most OpenAI SDKs (Python, JS, etc.) — just change the base URL to `http://localhost:1234/v1` (or your custom port/IP).
- You can also serve it over your local network (enable "Serve on Local Network" in settings) for other devices.

### Listing / Seeing the Active (or Loaded) Model
Yes, there are straightforward ways to list models and check what's active/loaded:

1. **Via OpenAI-compatible endpoint** (easiest for most integrations):
   - `GET http://localhost:1234/v1/models`
   - This returns a standard OpenAI-style response listing available models.
   - In many setups (especially with Just-In-Time / auto-loading enabled), it shows all downloaded models, but the "active" one is typically the one currently loaded for inference.
   - You can curl it directly or use any OpenAI client to call `client.models.list()` after pointing to LM Studio.

2. **Via LM Studio's native REST API** (more detailed for management):
   - `GET http://localhost:1234/api/v1/models` — Lists all available models on your system (downloaded LLMs + embedding models), including their state (e.g., "loaded" vs "not-loaded").
   - This explicitly shows loading status, quantization, context length, etc.
   - There's also `GET /api/v1/models/{model}` for details on a specific one, and endpoints like `POST /api/v1/models/load` and `POST /api/v1/models/unload` to control what's active.

In the app UI itself, you can always see the currently loaded/active model right in the chat interface or model selector — it shows what's ready for chatting/inference.

If you're building something programmatic (e.g., scripts, agents, or apps talking to LM Studio), the `/v1/models` endpoint is usually sufficient and the most portable. For full control over loading/unloading, use the native `/api/v1/models` family.

The docs are at lmstudio.ai/docs (especially /developer/openai-compat and /developer/rest) if you want the exact schemas and examples. Let me know if you're trying to integrate it with a specific language/SDK or use case!