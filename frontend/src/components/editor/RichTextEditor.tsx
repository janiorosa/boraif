import { useEffect } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import type { Editor, JSONContent } from "@tiptap/core";
import StarterKit from "@tiptap/starter-kit";
import Underline from "@tiptap/extension-underline";
import Link from "@tiptap/extension-link";
import TextAlign from "@tiptap/extension-text-align";
import Subscript from "@tiptap/extension-subscript";
import Superscript from "@tiptap/extension-superscript";
import Image from "@tiptap/extension-image";
import { TableKit } from "@tiptap/extension-table";
import { Mathematics } from "@tiptap/extension-mathematics";
import { EditorToolbar } from "./EditorToolbar";
import "katex/dist/katex.min.css";
import "./editor.css";

interface RichTextEditorProps {
  content: unknown;
  onChange: (json: unknown) => void;
  disciplineId: number;
  minHeight?: number;
  // Expõe a instância do editor para o pai poder chamar editor.getText()
  // sob demanda (ex.: extrair texto simples para o assistente de IA —
  // seção 16) sem precisar duplicar esse estado em toda tecla digitada.
  onEditorReady?: (editor: Editor) => void;
}

// Componente único usado para enunciado, comando e as cinco alternativas
// (seção 9) — mesmas ferramentas em todos, o que muda é só qual campo cada
// instância representa. `content` só é usado na criação do editor; edições
// posteriores vivem dentro do próprio editor e saem via onChange, nunca o
// contrário (evita que o editor "pule" enquanto o professor digita).
export function RichTextEditor({ content, onChange, disciplineId, minHeight = 60, onEditorReady }: RichTextEditorProps) {
  const editor = useEditor({
    immediatelyRender: false,
    shouldRerenderOnTransaction: true,
    extensions: [
      StarterKit.configure({ heading: false }),
      Underline,
      Link.configure({ openOnClick: false, autolink: false }),
      TextAlign.configure({ types: ["paragraph"] }),
      Subscript,
      Superscript,
      Image,
      TableKit.configure({ table: { resizable: false } }),
      Mathematics,
    ],
    content: content as JSONContent,
    onUpdate: ({ editor }) => onChange(editor.getJSON()),
  });

  useEffect(() => {
    if (editor) onEditorReady?.(editor);
  }, [editor, onEditorReady]);

  if (!editor) return null;

  return (
    <div style={{ border: "1px solid #ccc", borderRadius: 4 }}>
      <EditorToolbar editor={editor} disciplineId={disciplineId} />
      <div style={{ minHeight, padding: 8 }}>
        <EditorContent editor={editor} />
      </div>
    </div>
  );
}
