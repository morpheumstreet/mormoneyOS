**Yes, Jan AI has a built-in server API** — specifically, a **local OpenAI-compatible API server** that runs entirely on your computer (powered by llama.cpp or similar backends in recent versions).

You enable it in the app settings (usually under something like "Local API Server" or "Server" in the settings panel), and once started, it listens by default at `http://127.0.0.1:1337` (or `localhost:1337`).

It supports standard OpenAI-style endpoints like:
- `/v1/chat/completions` (for chat)
- `/v1/models` (lists available models — more on this below)
- And others following OpenAI's spec (with some limitations noted in older issues, but it's generally very compatible for most use cases like integrations, scripting, or tools like Continue.dev).

You can test it easily with curl, for example:
```bash
curl http://127.0.0.1:1337/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer secret-key-123" \
  -d '{
    "model": "your-model-id",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

(Note: The auth token is often a dummy/fixed one like "secret-key-123" or configurable; check Jan's docs or logs.)

Regarding **listing and seeing the active/currently loaded model**:

- The standard OpenAI endpoint `GET /v1/models` **does exist** and returns a list of **available** models (those downloaded/installed in Jan's hub).
- However, it lists **all** models Jan knows about, **not necessarily only the one actively loaded** in memory for inference right now.
- There's an open feature request / discussion in their GitHub (from late 2024) about adding a way to specifically query the **currently loaded model(s)** via API (since `/v1/models` shows availability, not active runtime state).
- In practice:
  - If you're only running one model at a time (common in desktop use), the active one is usually the last one you selected in the UI or the one you specify in API requests.
  - Jan's desktop app UI shows the current active model clearly in the chat/model selector.
  - For programmatic lookup of the "active" one, there isn't a dedicated endpoint yet (as of early 2026 docs), but you can infer it from recent chats or by attempting inferences and seeing which succeeds without loading delay.

There's also a separate **Jan Server** project (enterprise/self-hosted variant), which has more advanced API features (including better model management and tool orchestration), but for the standard desktop Jan app, the local API server is what's built-in and most people use.

Overall:  
- Server API → **Yes, excellent OpenAI drop-in** (very similar to LM Studio or Ollama's compatibility).  
- List active model via API → **Partially** (you get available ones easily; strictly "active/loaded" is more UI-visible or requires workarounds).

Check the official docs for the latest (they update frequently): https://www.jan.ai/docs/desktop/api-server

If you're integrating it into something specific (VS Code extension, script, etc.), let me know — I can help with example code!