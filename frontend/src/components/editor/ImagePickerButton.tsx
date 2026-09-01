import { useRef, useState, type ChangeEvent } from "react";
import type { Editor } from "@tiptap/core";
import { api, ApiError } from "../../api/client";
import { ImageGrid } from "../../pages/images/ImageGrid";
import type { ImageItem } from "../../pages/images/types";

interface Props {
  editor: Editor;
  disciplineId: number;
}

type Mode = "upload" | "library";

// Um único botão de imagem cobre os dois lados da seção 13: enviar um
// arquivo novo, ou reaproveitar uma imagem que outro professor da mesma
// disciplina já enviou (biblioteca — mesma origem de dados da página
// dedicada /imagens).
export function ImagePickerButton({ editor, disciplineId }: Props) {
  const [open, setOpen] = useState(false);
  const [mode, setMode] = useState<Mode>("upload");
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function insert(url: string) {
    editor.chain().focus().setImage({ src: url }).run();
    setOpen(false);
  }

  async function handleFileChange(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;

    setUploading(true);
    setError(null);
    try {
      const formData = new FormData();
      formData.append("file", file);
      formData.append("disciplineId", String(disciplineId));
      const result = await api.upload<{ id: number; url: string }>("/api/images", formData);
      insert(result.url);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Não foi possível enviar a imagem.");
    } finally {
      setUploading(false);
    }
  }

  return (
    <>
      <button type="button" title="Inserir imagem" onClick={() => setOpen(true)}>
        🖼
      </button>

      {open && (
        <div
          style={{
            position: "fixed",
            inset: 0,
            background: "rgba(0,0,0,0.3)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex: 1000,
          }}
          onClick={() => setOpen(false)}
        >
          <div
            style={{ background: "white", padding: 16, borderRadius: 4, width: 420 }}
            onClick={(e) => e.stopPropagation()}
          >
            <h3 style={{ marginTop: 0 }}>Inserir imagem</h3>

            <div style={{ display: "flex", gap: 8, marginBottom: 12 }}>
              <button
                type="button"
                onClick={() => setMode("upload")}
                style={{ fontWeight: mode === "upload" ? "bold" : "normal" }}
              >
                Enviar nova
              </button>
              <button
                type="button"
                onClick={() => setMode("library")}
                style={{ fontWeight: mode === "library" ? "bold" : "normal" }}
              >
                Escolher da biblioteca
              </button>
            </div>

            {mode === "upload" ? (
              <div>
                <button type="button" onClick={() => inputRef.current?.click()} disabled={uploading}>
                  {uploading ? "Enviando..." : "Selecionar arquivo"}
                </button>
                <input
                  ref={inputRef}
                  type="file"
                  accept="image/png,image/jpeg,image/gif,image/webp"
                  hidden
                  onChange={handleFileChange}
                />
                {error && <p style={{ color: "crimson" }}>{error}</p>}
              </div>
            ) : (
              <ImageGrid disciplineId={disciplineId} onSelect={(img: ImageItem) => insert(img.url)} />
            )}

            <div style={{ display: "flex", justifyContent: "flex-end", marginTop: 12 }}>
              <button type="button" onClick={() => setOpen(false)}>
                Fechar
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
