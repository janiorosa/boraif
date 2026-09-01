// Documento ProseMirror vazio, usado só para popular enunciado/comando/
// alternativas de uma questão recém-criada, antes do professor abrir o
// editor de verdade e começar a escrever (seção 37).
export function emptyDoc(): unknown {
  return { type: "doc", content: [{ type: "paragraph" }] };
}
