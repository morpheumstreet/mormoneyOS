import type { TreasuryPolicy } from "@/lib/api";

export const CHAIN_LABELS: Record<string, string> = {
  "eip155:8453": "Base",
  "eip155:84532": "Base Sepolia",
  "eip155:1": "Ethereum",
  "eip155:137": "Polygon",
  "eip155:42161": "Arbitrum",
};

export interface TreasuryFieldConfig {
  key: keyof TreasuryPolicy;
  label: string;
  defaultValue: number;
}

export const TREASURY_FIELDS: TreasuryFieldConfig[] = [
  { key: "maxSingleTransferCents", label: "Max single transfer (¢)", defaultValue: 5000 },
  { key: "maxHourlyTransferCents", label: "Max hourly transfer (¢)", defaultValue: 10000 },
  { key: "maxDailyTransferCents", label: "Max daily transfer (¢)", defaultValue: 50000 },
  { key: "minReserveCents", label: "Min reserve (¢)", defaultValue: 100 },
  {
    key: "inferenceDailyBudgetCents",
    label: "Inference daily budget (¢)",
    defaultValue: 5000,
  },
];
