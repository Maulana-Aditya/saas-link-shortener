import { useEffect, useState } from "react";
import { api, ApiError } from "../lib/api";
import { useAuth } from "../lib/auth";

type Usage = {
  plan: string;
  monthly_clicks: number;
  monthly_click_limit: number;
  link_limit: number;
};

export default function Billing() {
  const { org } = useAuth();
  const [usage, setUsage] = useState<Usage | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [redirecting, setRedirecting] = useState(false);

  useEffect(() => {
    api
      .get<Usage>("/api/v1/billing/usage")
      .then(setUsage)
      .catch((err) => setError(err instanceof ApiError ? err.message : "could not load usage"));
  }, []);

  const upgrade = async () => {
    setRedirecting(true);
    try {
      const { checkout_url } = await api.post<{ checkout_url: string }>("/api/v1/billing/checkout");
      window.location.href = checkout_url;
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "could not start checkout");
      setRedirecting(false);
    }
  };

  return (
    <div className="max-w-xl mx-auto space-y-6">
      <h1 className="text-2xl font-semibold">Billing</h1>
      {error && <p className="text-sm text-red-600">{error}</p>}

      <div className="border rounded-lg p-4 space-y-2">
        <p>
          Current plan: <span className="font-medium">{org?.plan ?? "..."}</span>
        </p>
        {usage && (
          <>
            <p className="text-sm text-gray-600">
              Links: plan limit {usage.link_limit} (free plan only)
            </p>
            <p className="text-sm text-gray-600">
              Clicks this period: {usage.monthly_clicks} / {usage.monthly_click_limit}
            </p>
          </>
        )}
      </div>

      {org?.plan === "free" && (
        <button
          onClick={upgrade}
          disabled={redirecting}
          className="bg-blue-600 text-white rounded px-4 py-2 disabled:opacity-50"
        >
          {redirecting ? "Redirecting to Stripe..." : "Upgrade to Pro"}
        </button>
      )}
    </div>
  );
}
