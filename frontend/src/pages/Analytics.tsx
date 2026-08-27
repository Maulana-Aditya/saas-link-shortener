import { useEffect, useState } from "react";
import { api, ApiError } from "../lib/api";

type DailyCount = { day: string; count: number };
type TopReferrer = { referrer: string; count: number };
type Summary = { clicks_by_day: DailyCount[] | null; top_referrers: TopReferrer[] | null };

export default function Analytics() {
  const [summary, setSummary] = useState<Summary | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .get<Summary>("/api/v1/analytics/summary?days=30")
      .then(setSummary)
      .catch((err) => setError(err instanceof ApiError ? err.message : "could not load analytics"));
  }, []);

  const byDay = summary?.clicks_by_day ?? [];
  const referrers = summary?.top_referrers ?? [];
  const maxCount = Math.max(1, ...byDay.map((d) => d.count));

  return (
    <div className="max-w-3xl mx-auto space-y-8">
      <h1 className="text-2xl font-semibold">Analytics (last 30 days)</h1>
      {error && <p className="text-sm text-red-600">{error}</p>}

      <section>
        <h2 className="font-medium mb-2">Clicks by day</h2>
        {byDay.length === 0 ? (
          <p className="text-gray-500 text-sm">No clicks recorded yet.</p>
        ) : (
          <div className="flex items-end gap-1 h-32 border-b">
            {byDay.map((d) => (
              <div
                key={d.day}
                title={`${d.day}: ${d.count}`}
                className="bg-blue-500 w-3 rounded-t"
                style={{ height: `${(d.count / maxCount) * 100}%` }}
              />
            ))}
          </div>
        )}
      </section>

      <section>
        <h2 className="font-medium mb-2">Top referrers</h2>
        {referrers.length === 0 ? (
          <p className="text-gray-500 text-sm">No referrer data yet.</p>
        ) : (
          <ul className="divide-y border rounded-lg">
            {referrers.map((r) => (
              <li key={r.referrer} className="flex justify-between px-4 py-2 text-sm">
                <span>{r.referrer}</span>
                <span className="text-gray-500">{r.count}</span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
