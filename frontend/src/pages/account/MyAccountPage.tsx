import { useEffect, useState, type FormEvent } from "react";
import { AppLayout } from "../../components/AppLayout";
import { api, ApiError } from "../../api/client";

// Seção 17: cada professor cadastra a própria API Key da OpenAI. A chave
// nunca é devolvida pelo backend depois de salva — só um status.
export function MyAccountPage() {
  const [configured, setConfigured] = useState<boolean | null>(null);
  const [apiKey, setApiKey] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    let cancelled = false;
    api.get<{ configured: boolean }>("/api/me/openai-key/status").then((res) => {
      if (!cancelled) setConfigured(res.configured);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      await api.put("/api/me/openai-key", { apiKey });
      setConfigured(true);
      setApiKey("");
      setSaved(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Não foi possível salvar a chave.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <AppLayout>
      <h1>Minha conta</h1>
      <h2>API Key da OpenAI</h2>
      <p>
        Status:{" "}
        {configured === null ? (
          "carregando..."
        ) : configured ? (
          <strong style={{ color: "green" }}>configurada</strong>
        ) : (
          <strong style={{ color: "crimson" }}>não configurada</strong>
        )}
      </p>
      <p style={{ color: "#666", maxWidth: 480 }}>
        Necessária para usar o assistente de revisão por IA no editor de questões. Sua chave é armazenada cifrada e
        nunca é exibida novamente depois de salva — só este status.
      </p>

      <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: 8, maxWidth: 400 }}>
        <label>
          {configured ? "Substituir chave" : "Cadastrar chave"}
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="sk-..."
            required
            style={{ width: "100%" }}
          />
        </label>
        {error && <p style={{ color: "crimson" }}>{error}</p>}
        {saved && <p style={{ color: "green" }}>Chave salva.</p>}
        <button type="submit" disabled={saving}>
          {saving ? "Salvando..." : "Salvar"}
        </button>
      </form>
    </AppLayout>
  );
}
