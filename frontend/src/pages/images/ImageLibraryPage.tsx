import { useEffect, useRef, useState, type ChangeEvent } from "react";
import { AppLayout } from "../../components/AppLayout";
import { ImageGrid } from "./ImageGrid";
import { api, ApiError } from "../../api/client";
import { useAuth } from "../../auth/AuthContext";
import type { Discipline } from "../../types";

// Área "Imagens" da interface principal (seção 36): biblioteca de imagens
// da disciplina, compartilhada entre todos os professores dela (seção 13).
export function ImageLibraryPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === "ADMIN";

  const [disciplines, setDisciplines] = useState<Discipline[]>([]);
  const [disciplineId, setDisciplineId] = useState<number | "">(isAdmin ? "" : user?.disciplineId ?? "");
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!isAdmin) return;
    let cancelled = false;
    api.get<Discipline[]>("/api/disciplines").then((list) => {
      if (!cancelled) setDisciplines(list);
    });
    return () => {
      cancelled = true;
    };
  }, [isAdmin]);

  async function handleUpload(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file || disciplineId === "") return;

    setUploading(true);
    setUploadError(null);
    try {
      const formData = new FormData();
      formData.append("file", file);
      formData.append("disciplineId", String(disciplineId));
      await api.upload("/api/images", formData);
      setRefreshKey((k) => k + 1);
    } catch (err) {
      setUploadError(err instanceof ApiError ? err.message : "Não foi possível enviar a imagem.");
    } finally {
      setUploading(false);
    }
  }

  return (
    <AppLayout>
      <h1>Imagens</h1>

      {isAdmin && (
        <label>
          Disciplina:{" "}
          <select
            value={disciplineId}
            onChange={(e) => setDisciplineId(e.target.value === "" ? "" : Number(e.target.value))}
          >
            <option value="" disabled>
              Selecione...
            </option>
            {disciplines.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </label>
      )}

      {disciplineId === "" ? (
        <p style={{ color: "#666" }}>Selecione uma disciplina para ver a biblioteca de imagens.</p>
      ) : (
        <>
          <div style={{ margin: "12px 0" }}>
            <button type="button" onClick={() => inputRef.current?.click()} disabled={uploading}>
              {uploading ? "Enviando..." : "Enviar imagem"}
            </button>
            <input
              ref={inputRef}
              type="file"
              accept="image/png,image/jpeg,image/gif,image/webp"
              hidden
              onChange={handleUpload}
            />
            {uploadError && <span style={{ color: "crimson", marginLeft: 8 }}>{uploadError}</span>}
          </div>
          <ImageGrid key={refreshKey} disciplineId={disciplineId} />
        </>
      )}
    </AppLayout>
  );
}
