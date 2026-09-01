import { useState, type ReactNode } from "react";
import type { Editor } from "@tiptap/core";
import { FormulaDialog } from "./FormulaDialog";
import { ImagePickerButton } from "./ImagePickerButton";

interface Props {
  editor: Editor;
  disciplineId: number;
}

// Barra de ferramentas única, compartilhada por enunciado, comando e pelas
// cinco alternativas — os recursos são os da seção 10, sem transformar o
// editor num processador de texto completo.
export function EditorToolbar({ editor, disciplineId }: Props) {
  const [formulaOpen, setFormulaOpen] = useState(false);

  function setLink() {
    const previousUrl = (editor.getAttributes("link").href as string | undefined) ?? "";
    const url = window.prompt("URL do link:", previousUrl);
    if (url === null) return;
    if (url === "") {
      editor.chain().focus().unsetLink().run();
      return;
    }
    editor.chain().focus().setLink({ href: url }).run();
  }

  function insertTable() {
    editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run();
  }

  return (
    <div style={{ display: "flex", flexWrap: "wrap", alignItems: "center", gap: 2, borderBottom: "1px solid #ccc", padding: 4 }}>
      <ToolbarButton active={editor.isActive("bold")} onClick={() => editor.chain().focus().toggleBold().run()} title="Negrito">
        <strong>N</strong>
      </ToolbarButton>
      <ToolbarButton active={editor.isActive("italic")} onClick={() => editor.chain().focus().toggleItalic().run()} title="Itálico">
        <em>I</em>
      </ToolbarButton>
      <ToolbarButton
        active={editor.isActive("underline")}
        onClick={() => editor.chain().focus().toggleUnderline().run()}
        title="Sublinhado"
      >
        <u>S</u>
      </ToolbarButton>
      <ToolbarButton active={editor.isActive("strike")} onClick={() => editor.chain().focus().toggleStrike().run()} title="Tachado">
        <s>T</s>
      </ToolbarButton>
      <ToolbarButton
        active={editor.isActive("subscript")}
        onClick={() => editor.chain().focus().toggleSubscript().run()}
        title="Subscrito"
      >
        X₂
      </ToolbarButton>
      <ToolbarButton
        active={editor.isActive("superscript")}
        onClick={() => editor.chain().focus().toggleSuperscript().run()}
        title="Sobrescrito"
      >
        X²
      </ToolbarButton>

      <Separator />

      <ToolbarButton
        active={editor.isActive("bulletList")}
        onClick={() => editor.chain().focus().toggleBulletList().run()}
        title="Lista"
      >
        •
      </ToolbarButton>
      <ToolbarButton
        active={editor.isActive("orderedList")}
        onClick={() => editor.chain().focus().toggleOrderedList().run()}
        title="Lista numerada"
      >
        1.
      </ToolbarButton>

      <Separator />

      <ToolbarButton active={editor.isActive({ textAlign: "left" })} onClick={() => editor.chain().focus().setTextAlign("left").run()} title="Alinhar à esquerda">
        ⟸
      </ToolbarButton>
      <ToolbarButton
        active={editor.isActive({ textAlign: "center" })}
        onClick={() => editor.chain().focus().setTextAlign("center").run()}
        title="Centralizar"
      >
        ⟺
      </ToolbarButton>
      <ToolbarButton
        active={editor.isActive({ textAlign: "right" })}
        onClick={() => editor.chain().focus().setTextAlign("right").run()}
        title="Alinhar à direita"
      >
        ⟹
      </ToolbarButton>

      <Separator />

      <ToolbarButton active={editor.isActive("link")} onClick={setLink} title="Link">
        🔗
      </ToolbarButton>
      <ImagePickerButton editor={editor} disciplineId={disciplineId} />
      <ToolbarButton onClick={insertTable} title="Inserir tabela">
        ▦
      </ToolbarButton>
      <ToolbarButton onClick={() => setFormulaOpen(true)} title="Fórmula">
        ∑
      </ToolbarButton>

      <Separator />

      <ToolbarButton onClick={() => editor.chain().focus().undo().run()} title="Desfazer">
        ↶
      </ToolbarButton>
      <ToolbarButton onClick={() => editor.chain().focus().redo().run()} title="Refazer">
        ↷
      </ToolbarButton>

      {formulaOpen && (
        <FormulaDialog
          onInsert={(latex) => {
            editor.chain().focus().insertInlineMath({ latex }).run();
            setFormulaOpen(false);
          }}
          onClose={() => setFormulaOpen(false)}
        />
      )}
    </div>
  );
}

function ToolbarButton({ active, onClick, title, children }: { active?: boolean; onClick: () => void; title: string; children: ReactNode }) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={title}
      style={{
        minWidth: 28,
        border: "1px solid #ccc",
        borderRadius: 3,
        background: active ? "#ddd" : "white",
        cursor: "pointer",
      }}
    >
      {children}
    </button>
  );
}

function Separator() {
  return <span style={{ width: 1, alignSelf: "stretch", background: "#ccc", margin: "0 4px" }} />;
}
