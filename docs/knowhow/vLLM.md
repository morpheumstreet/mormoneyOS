**Yes, vLLM's OpenAI-compatible server includes a `/v1/models` endpoint that lets you list (and effectively view details about) the currently loaded/active model(s).**

This follows the standard OpenAI API spec for listing models (GET `/v1/models`), which returns a list of available models with basic metadata like ID, creation timestamp, and owner.

### Key Details
- **Endpoint**: `GET /v1/models`
- **Default URL** (when running locally): `http://localhost:8000/v1/models`
- **What it returns**: Since vLLM typically serves **one primary model per server instance** (the one you specify with `--model` or in `vllm serve <model>`), the response is usually a simple list with that single model.
- **Example response** (JSON):
  ```json
  {
    "object": "list",
    "data": [
      {
        "id": "meta-llama/Llama-3.1-8B-Instruct",
        "object": "model",
        "created": 1720000000,
        "owned_by": "vllm"
      }
    ]
  }
  ```
  - The `id` is usually the model name/path you loaded (or an override if you set one).
  - In multi-model or router setups (e.g., via wrappers or forks), it might list more, but in standard vLLM it's single-model focused.

### How to Query It
- **With curl** (quick test):
  ```bash
  curl http://localhost:8000/v1/models
  ```
  (Add `-H "Authorization: Bearer $VLLM_API_KEY"` if you started the server with `--api-key` or `VLLM_API_KEY` env var.)

- **With OpenAI Python client**:
  ```python
  from openai import OpenAI

  client = OpenAI(
      base_url="http://localhost:8000/v1",
      api_key="EMPTY"  # or your actual key if auth is enabled
  )

  models = client.models.list()
  print(models.data)  # List of model objects
  print(models.data[0].id)  # e.g. your loaded model name
  ```

### Notes & Limitations (as of 2026)
- vLLM's server is optimized for **high-throughput serving of one main model** (or a few in advanced configs), so `/v1/models` is simple and doesn't show "active sessions" or detailed runtime stats — just the loaded model(s).
- No built-in endpoint for "view detailed status" like memory usage, active requests, or live metrics (for that, people often use Prometheus metrics endpoint `--enable-metrics` + `/metrics`, or external monitoring).
- If you're running multiple models, you'd typically spin up separate vLLM instances (different ports) or use a proxy/router in front (e.g., LiteLLM, semantic-router, or nginx).
- The endpoint has been supported since early versions and is explicitly mentioned in the official docs (e.g., Quickstart guide shows `curl http://localhost:8000/v1/models` as an example).

Compared to llama.cpp (which also has `/v1/models` with similar single-model output) or LocalAI (more multi-model friendly with richer listings), vLLM keeps it minimal but fully compatible for most tools that expect OpenAI-style model discovery.

If you need something more advanced (e.g., health checks, detailed engine stats), let me know your setup — I can suggest flags or sidecar tools!