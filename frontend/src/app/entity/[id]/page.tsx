import { redirect } from "next/navigation";

export default async function EntityPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  redirect(`/leads/${encodeURIComponent(id)}`);
}
