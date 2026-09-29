"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { copyToClipboard, truncateHash } from "@/lib/format";

export function HashValue({
  value,
  truncate = true,
  lead = 8,
  tail = 6,
  className = "",
}: {
  value: string;
  truncate?: boolean;
  lead?: number;
  tail?: number;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    const ok = await copyToClipboard(value);
    if (ok) {
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    }
  }

  return (
    <button
      type="button"
      onClick={handleCopy}
      title={value}
      className={`focus-ring group mono-id inline-flex max-w-full items-center gap-1.5 text-left text-text-secondary hover:text-text ${className}`}
    >
      <span className="truncate">{truncate ? truncateHash(value, lead, tail) : value}</span>
      {copied ? (
        <Check className="h-3 w-3 shrink-0 text-success" />
      ) : (
        <Copy className="h-3 w-3 shrink-0 opacity-0 group-hover:opacity-60" />
      )}
    </button>
  );
}
