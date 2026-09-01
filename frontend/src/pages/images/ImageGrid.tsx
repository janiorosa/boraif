import { useEffect, useState } from "react";
import { api } from "../../api/client";
import type { ImageItem } from "./types";

interface Props {
  disciplineId: number;
  onSelect?: (image: ImageItem) => void;
}

// Grid de busca/reutilização de imagens da disciplina (seção 13/36) —
// usado tanto na página dedicada (/imagens) quanto no seletor "escolher da
// biblioteca" dentro do próprio editor de questões.
export function ImageGrid({ disciplineId, onSelect }: Props) {
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [items, setItems] = useState<ImageItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 400);
    return () => clearTimeout(timer);
  }, [search]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    const params = new URLSearchParams({ disciplineId: String(disciplineId) });
    if (debouncedSearch) params.set("search", debouncedSearch);

    api
      .get<{ items: ImageItem[]; total: number }>(`/api/images?${params.toString()}`)
      .then((result) => {
        if (!cancelled) setItems(result.items);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [disciplineId, debouncedSearch]);

  return (
    <div>
      <input
        placeholder="Buscar por nome do arquivo..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        style={{ marginBottom: 8, width: "100%" }}
      />
      {loading ? (
        <p>Carregando...</p>
      ) : items.length === 0 ? (
        <p style={{ color: "#666" }}>Nenhuma imagem encontrada.</p>
      ) : (
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(100px, 1fr))", gap: 8 }}>
          {items.map((img) => (
            <button
              key={img.id}
              type="button"
              onClick={() => (onSelect ? onSelect(img) : window.open(img.url, "_blank"))}
              title={img.filename}
              style={{
                padding: 0,
                border: "1px solid #ccc",
                borderRadius: 4,
                overflow: "hidden",
                cursor: "pointer",
                background: "none",
              }}
            >
              <img
                src={img.url}
                alt={img.filename}
                style={{ width: "100%", height: 80, objectFit: "cover", display: "block" }}
              />
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
