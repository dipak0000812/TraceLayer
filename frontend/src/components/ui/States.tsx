import { ReactNode } from "react";
import Link from "next/link";

export function EmptyState({
  title,
  description,
  actionHref,
  actionLabel,
}: {
  title: string;
  description: string;
  actionHref?: string;
  actionLabel?: string;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 px-6 py-14 text-center">
      <p className="text-sm font-medium text-text">{title}</p>
      <p className="max-w-sm text-sm text-text-muted">{description}</p>
      {actionHref && actionLabel && (
        <Link
          href={actionHref}
          className="focus-ring mt-2 rounded-md border border-border-strong bg-surface-active px-3 py-1.5 text-xs font-medium text-text hover:bg-surface-hover"
        >
          {actionLabel}
        </Link>
      )}
    </div>
  );
}

export function ErrorState({
  title,
  message,
  detail,
  onRetry,
}: {
  title: string;
  message: string;
  detail?: string;
  onRetry?: () => void;
}) {
  return (
    <div className="flex flex-col gap-2 rounded-lg border border-danger/25 bg-danger-muted/40 px-5 py-4">
      <div className="flex items-center justify-between gap-4">
        <div>
          <p className="text-sm font-medium text-text">{title}</p>
          <p className="mt-0.5 text-sm text-text-secondary">{message}</p>
        </div>
        {onRetry && (
          <button
            onClick={onRetry}
            className="focus-ring shrink-0 rounded-md border border-border-strong bg-surface px-3 py-1.5 text-xs font-medium text-text hover:bg-surface-hover"
          >
            Retry
          </button>
        )}
      </div>
      {detail && (
        <details className="text-xs text-text-muted">
          <summary className="cursor-pointer select-none hover:text-text-secondary">
            Details
          </summary>
          <pre className="mono-id mt-1 whitespace-pre-wrap break-all text-2xs text-text-muted">
            {detail}
          </pre>
        </details>
      )}
    </div>
  );
}

export function Panel({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={`rounded-lg border border-border bg-surface ${className}`}>
      {children}
    </div>
  );
}

export function PanelHeader({
  title,
  subtitle,
  action,
}: {
  title: string;
  subtitle?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex items-center justify-between border-b border-border px-5 py-3.5">
      <div>
        <h3 className="text-sm font-medium text-text">{title}</h3>
        {subtitle && <p className="mt-0.5 text-xs text-text-muted">{subtitle}</p>}
      </div>
      {action}
    </div>
  );
}

export function Skeleton({ className = "" }: { className?: string }) {
  return (
    <div className={`animate-pulse rounded bg-surface-active ${className}`} />
  );
}
