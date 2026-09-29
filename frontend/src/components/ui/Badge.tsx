import { ReactNode } from "react";

type Tone = "neutral" | "success" | "warning" | "danger" | "info";

const toneClasses: Record<Tone, string> = {
  neutral: "bg-surface-active text-text-secondary border-border-strong",
  success: "bg-success-muted text-success border-success/30",
  warning: "bg-warning-muted text-warning border-warning/30",
  danger: "bg-danger-muted text-danger border-danger/30",
  info: "bg-accent-muted text-accent-strong border-accent/30",
};

export function Badge({
  children,
  tone = "neutral",
  mono = false,
}: {
  children: ReactNode;
  tone?: Tone;
  mono?: boolean;
}) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-2xs font-medium leading-none ${
        mono ? "mono-id" : ""
      } ${toneClasses[tone]}`}
    >
      {children}
    </span>
  );
}

export function StatusDot({ tone = "neutral" }: { tone?: Tone }) {
  const dot: Record<Tone, string> = {
    neutral: "bg-text-muted",
    success: "bg-success",
    warning: "bg-warning",
    danger: "bg-danger",
    info: "bg-accent",
  };
  return <span className={`inline-block h-1.5 w-1.5 rounded-full ${dot[tone]}`} />;
}
