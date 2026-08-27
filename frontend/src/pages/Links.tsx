import { useEffect, useState, type FormEvent } from "react";
import { api, ApiError } from "../lib/api";

type LinkItem = {
  id: string;
  slug: string;
  long_url: string;
  short_url: string;
};

export default function Links() {
  const [links, setLinks] = useState<LinkItem[]>([]);
  const [longUrl, setLongUrl] = useState("");
  const [customSlug, setCustomSlug] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const data = await api.get<{ links: LinkItem[] | null }>("/api/v1/links");
      setLinks(data.links ?? []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "could not load links");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await api.post("/api/v1/links", { long_url: longUrl, slug: customSlug || undefined });
      setLongUrl("");
      setCustomSlug("");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "could not create link");
    } finally {
      setSubmitting(false);
    }
  };

  const onDelete = async (id: string) => {
    await api.delete(`/api/v1/links/${id}`);
    await load();
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-semibold">Your links</h1>

      <form onSubmit={onSubmit} className="flex flex-wrap gap-2 border rounded-lg p-4">
        <input
          type="url"
          required
          placeholder="https://example.com/very/long/path"
          value={longUrl}
          onChange={(e) => setLongUrl(e.target.value)}
          className="flex-1 min-w-[240px] border rounded px-3 py-2"
        />
        <input
          placeholder="custom-slug (optional)"
          value={customSlug}
          onChange={(e) => setCustomSlug(e.target.value)}
          className="w-48 border rounded px-3 py-2"
        />
        <button
          type="submit"
          disabled={submitting}
          className="bg-blue-600 text-white rounded px-4 py-2 disabled:opacity-50"
        >
          Shorten
        </button>
      </form>
      {error && <p className="text-sm text-red-600">{error}</p>}

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : links.length === 0 ? (
        <p className="text-gray-500">No links yet — create your first one above.</p>
      ) : (
        <ul className="divide-y border rounded-lg">
          {links.map((l) => (
            <li key={l.id} className="flex items-center justify-between p-4">
              <div>
                <a
                  href={l.short_url}
                  target="_blank"
                  rel="noreferrer"
                  className="font-medium text-blue-600"
                >
                  {l.short_url}
                </a>
                <p className="text-sm text-gray-500 truncate max-w-md">{l.long_url}</p>
              </div>
              <button onClick={() => onDelete(l.id)} className="text-red-600 text-sm hover:underline">
                Delete
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
