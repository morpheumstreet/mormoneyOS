export const MASKED_PLACEHOLDER = "••••••••";

export const PROVIDER_LABELS: Record<string, string> = {
  bore: "Bore",
  localtunnel: "Localtunnel",
  cloudflare: "Cloudflare",
  ngrok: "ngrok",
  tailscale: "Tailscale",
  custom: "Custom",
};

export const PROVIDERS_WITH_RESTART = ["cloudflare", "ngrok", "tailscale"];
