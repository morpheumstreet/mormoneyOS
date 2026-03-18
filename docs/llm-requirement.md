**final, practical setup requirement guide** for installing and running **mormOS (mormoneyOS)** from github.com/morpheumstreet/mormoneyOS, with a strong focus on the **LLM requirements** to avoid common issues like the "chatjimmy returned empty response (input may exceed prefill limit ~6k tokens)" error you've seen. This error occurs when the LLM backend rejects or silently fails on prompts that exceed its effective context/prefill limit (even after mormOS's internal truncation to system + memory + input).

mormOS is a Docker-based autonomous agent system for investment/trading/research with model routing, memory tiers, and reflection. It does **not** hardcode a specific LLM — it calls an **OpenAI-compatible API endpoint** (local or remote).

### 1. Hardware Requirements (for reliable 24/7 local LLM + swarms)
To run meaningfully large models (needed for good investment reasoning/prediction without frequent truncation failures):

- **Best overall (recommended for HK 24/7, silent/cool/efficient)**:  
  Apple Mac Studio M4 Max (64 GB unified memory minimum, 128 GB ideal)  
  → Excellent for MLX/Ollama, handles 128k+ context KV cache easily, low power (~150–200 W load), silent fans.

- **Good compact/budget alternative**: Mac Mini M4 Pro (48–64 GB unified memory).

- **Max performance (if you accept higher power/heat)**:  
  PC with AMD Ryzen 9 9950X + NVIDIA RTX 5090 (32 GB GDDR7 VRAM) or RTX 4090/5090 equivalent.

- **Minimum viable (still hits truncation sometimes on complex agents)**: 32 GB unified/RAM + 16–24 GB VRAM/GPU.

Run https://www.canirun.ai/ in your browser on the target machine — look for your chosen model to show **"Runs great"** or at least **"Tight fit"** with plenty of headroom.

### 2. LLM Requirements & Best Choices (2026)
mormOS needs an **OpenAI-compatible server** (Chat Completions endpoint). Use **local** for sovereignty/no costs.

**Top recommendation (fixes truncation + best for finance/prediction/swarm scale):**
- **Qwen 3.5 35B-A3B MoE** (Q4_K_M or Q5_K_M GGUF quant)  
  → Native **262k context** (extendable), MoE efficiency (~3B active params → low power/heat for 24/7), excellent math/reasoning/tool-use/finance performance.

**Very strong alternative (deeper pure reasoning):**
- **DeepSeek-R1 Distill 14B** or **32B** (Q4_K_M / Q5_K_M)  
  → Often #1 in 2026 finance/quant benchmarks, 128k+ context.

**Smaller fallback (if hardware limited):**
- Qwen 3.5 14B / 9B or DeepSeek-R1 Distill 7B.

Avoid old/small models (<128k native context) or dense 70B+ without huge VRAM — they trigger prefill/context rejections.

### 3. Step-by-Step Installation & Setup

**Step 1: Install Docker**  
- macOS: Download Docker Desktop from https://www.docker.com/products/docker-desktop  
- Windows/Linux: Follow official Docker install guide.  
(Required — mormOS runs entirely in Docker.)

**Step 2: One-Line Install mormOS**  
Open terminal and run:

```bash
curl -fsSL https://raw.githubusercontent.com/morpheumstreet/mormoneyOS/main/scripts/install-docker.sh | bash
```

This:
- Pulls the latest image
- Creates/mounts `~/.automaton` for persistent data (wallets, memory, positions, logs)
- Starts the container
- Exposes web dashboard at http://localhost:8080

For background/daemon mode (recommended 24/7):

```bash
MORMONEYOS_DAEMON=1 curl -fsSL https://raw.githubusercontent.com/morpheumstreet/mormoneyOS/main/scripts/install-docker.sh | bash
```

Multi-bot example (e.g. separate trading/research agents):

```bash
MORMONEYOS_BOT=trading MORMONEYOS_PORT=8080 MORMONEYOS_DAEMON=1 curl ... | bash
MORMONEYOS_BOT=research MORMONEYOS_PORT=8081 MORMONEYOS_DAEMON=1 curl ... | bash
```

**Step 3: Set Up Local LLM Backend (Critical for avoiding empty response errors)**  
Use **Ollama** (easiest on Mac/PC, Apple Silicon optimized).

1. Install Ollama: https://ollama.com/download  
   macOS: `brew install ollama` (if Homebrew) or direct download.

2. Start Ollama server:  
   ```bash
   ollama serve
   ```
   (Keep this running — or run as service.)

3. Pull & create long-context version of the model:

   For **Qwen 3.5 35B-A3B MoE** (recommended):

   ```bash
   ollama pull qwen2.5:32b   # or exact 35B-A3B tag if available; fallback to closest MoE variant
   ```

   Create custom Modelfile for high context (fixes ~6k prefill wall):

   Save as `Modelfile`:

   ```
   FROM qwen2.5:32b
   PARAMETER num_ctx 131072          # or 262144 if model supports
   PARAMETER num_predict -1
   PARAMETER temperature 0.7
   ```

   Then:

   ```bash
   ollama create qwen-long-35b -f Modelfile
   ```

   Run/test it:

   ```bash
   ollama run qwen-long-35b
   ```

   Alternative for **DeepSeek-R1 Distill**:

   ```bash
   ollama pull deepseek-r1:14b   # or :32b distill variant
   ```

   Use similar Modelfile with `num_ctx 131072`.

**Step 4: Connect mormOS to Your Local LLM**  
Edit `~/.automaton/automaton.json` (created after first run):

Add/update the LLM section to point to Ollama (default OpenAI-compatible at http://localhost:11434):

```json
{
  "llm": {
    "provider": "openai",
    "base_url": "http://host.docker.internal:11434/v1",   // Mac: use host.docker.internal; Linux: localhost
    "api_key": "ollama",                                 // dummy key, Ollama ignores it
    "model": "qwen-long-35b"                             // your custom name
  },
  "TokenLimits": {
    "MaxInputTokens": 32000,                             // raise safely now
    "WarnAtTokens": 24000,
    "MaxHistoryTurns": 50
  }
}
```

- Save, restart mormOS container (docker restart or re-run install script).
- mormOS now uses long-context model → truncation to "system+memory+input only" almost never triggers.

**Step 5: Verify & Tune for 24/7**
- Open http://localhost:8080 — check dashboard/logs for agent turns without errors.
- Monitor GPU/CPU: `nvidia-smi` (PC) or Activity Monitor (Mac).
- If errors persist: Lower `MaxInputTokens` temporarily or check Ollama logs for OOM/context rejection.
- For MiroFish swarms: Configure in mormOS to route simple tasks to smaller/faster model if needed (via model routing in config).

**Summary — Final Recommended Stack (March 2026)**
- Hardware: Mac Studio M4 Max 64GB (or Mini M4 Pro 64GB)
- LLM: Qwen 3.5 35B-A3B MoE (Q4/Q5 quant, 128k+ context via custom Modelfile)
- Backend: Ollama (localhost:11434)
- mormOS: Latest via one-line Docker install + `automaton.json` tweaks

This setup gives strong financial reasoning, handles market data + memory + swarm state without hitting prefill limits, and runs truly 24/7 locally with minimal electricity/heat. If you hit issues after setup, share your `automaton.json` snippet or exact error logs — I can refine further. Your sovereign trading daemon is ready to go live.