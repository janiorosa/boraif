import { useEffect, useRef, useState } from "react";
import { useParams } from "react-router-dom";
import type { Editor } from "@tiptap/core";
import { AppLayout } from "../../components/AppLayout";
import { RichTextEditor } from "../../components/editor/RichTextEditor";
import { AIReviewPanel } from "./AIReviewPanel";
import { api, ApiError } from "../../api/client";
import type { Discipline } from "../../types";
import type { Subject } from "../subjects/types";
import type { AlternativePosition, Difficulty, GradeYear, QuestionDetail, QuestionStatus } from "./types";
import { ALL_POSITIONS } from "./constants";

type SaveState = "idle" | "saving" | "saved" | "error";

const AUTOSAVE_DEBOUNCE_MS = 2000;

// Tela de edição da questão (seção 9): enunciado, comando e as cinco
// alternativas são áreas TipTap independentes, cada uma com sua própria
// barra de ferramentas (formatação, imagens, tabelas, fórmulas — seção 10).
// A correta é indicada por um rádio ao lado de cada alternativa, nunca por
// uma sexta entidade (seção 7).
//
// Autosave (seção 18): salva o conjunto inteiro (metadados + enunciado +
// comando + 5 alternativas) num único PUT, nunca campo a campo, com
// debounce de 2s — nunca a cada tecla. Um botão "Salvar" manual convive
// com o autosave (critério de aceitação #13 e #14); os dois passam pelo
// mesmo performSave, que evita duas requisições simultâneas.
export function QuestionEditPage() {
  const { id } = useParams<{ id: string }>();
  const questionId = Number(id);

  const [question, setQuestion] = useState<QuestionDetail | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [disciplines, setDisciplines] = useState<Discipline[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [gradeYears, setGradeYears] = useState<GradeYear[]>([]);
  const [difficulties, setDifficulties] = useState<Difficulty[]>([]);
  const [statuses, setStatuses] = useState<QuestionStatus[]>([]);

  const [subjectId, setSubjectId] = useState(0);
  const [gradeYearId, setGradeYearId] = useState(0);
  const [difficultyId, setDifficultyId] = useState(0);
  const [statusId, setStatusId] = useState(0);
  const [statement, setStatement] = useState<unknown>(null);
  const [command, setCommand] = useState<unknown>(null);
  const [alternativeContents, setAlternativeContents] = useState<Record<AlternativePosition, unknown>>(
    {} as Record<AlternativePosition, unknown>,
  );
  const [correctPosition, setCorrectPosition] = useState<AlternativePosition>("A");

  const [saveState, setSaveState] = useState<SaveState>("idle");
  const [saveError, setSaveError] = useState<string | null>(null);

  // Instâncias do TipTap, guardadas só para o assistente de IA poder
  // extrair texto simples sob demanda (editor.getText()) ao clicar em um
  // dos botões de revisão — não usadas para renderização.
  const statementEditorRef = useRef<Editor | null>(null);
  const commandEditorRef = useRef<Editor | null>(null);
  const alternativeEditorRefs = useRef<Partial<Record<AlternativePosition, Editor>>>({});

  // Sempre com os valores mais recentes, para o autosave nunca mandar dado
  // velho mesmo se disparar em cima de um encadeamento de saves anteriores.
  const formRef = useRef({
    subjectId,
    gradeYearId,
    difficultyId,
    statusId,
    statement,
    command,
    alternativeContents,
    correctPosition,
  });
  formRef.current = {
    subjectId,
    gradeYearId,
    difficultyId,
    statusId,
    statement,
    command,
    alternativeContents,
    correctPosition,
  };

  // Precisa marcar true também no setup (não só declarar useRef(true)):
  // o StrictMode do React, em desenvolvimento, roda setup→cleanup→setup logo
  // na montagem para pegar efeitos mal limpos. Sem reafirmar aqui, o
  // cleanup do meio deixaria mountedRef permanentemente false, e
  // performSave nunca mais atualizaria a tela em dev.
  const mountedRef = useRef(true);
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  // Evita duas requisições de save simultâneas: se uma já está em voo
  // quando outra é pedida (autosave ou botão manual), a nova só roda depois
  // que a primeira termina, sempre com os dados mais atuais (seção 18 —
  // "a implementação deve evitar condições de corrida entre autosaves").
  const savingRef = useRef(false);
  const pendingRef = useRef(false);

  async function performSave() {
    if (!question) return;
    if (savingRef.current) {
      pendingRef.current = true;
      return;
    }
    savingRef.current = true;
    setSaveState("saving");
    setSaveError(null);

    const data = formRef.current;
    try {
      const alternatives = ALL_POSITIONS.map((position) => ({
        position,
        content: data.alternativeContents[position],
        isCorrect: position === data.correctPosition,
      }));
      const updated = await api.put<QuestionDetail>(`/api/questions/${question.id}`, {
        subjectId: data.subjectId,
        gradeYearId: data.gradeYearId,
        difficultyId: data.difficultyId,
        statusId: data.statusId,
        statement: data.statement,
        command: data.command,
        alternatives,
      });
      if (mountedRef.current) {
        setQuestion(updated);
        setSaveState("saved");
      }
    } catch (err) {
      if (mountedRef.current) {
        setSaveState("error");
        setSaveError(err instanceof ApiError ? err.message : "Não foi possível salvar.");
      }
    } finally {
      savingRef.current = false;
      if (pendingRef.current) {
        pendingRef.current = false;
        performSave();
      }
    }
  }

  useEffect(() => {
    let cancelled = false;
    api
      .get<QuestionDetail>(`/api/questions/${questionId}`)
      .then((q) => {
        if (cancelled) return;
        setQuestion(q);
        setSubjectId(q.subjectId);
        setGradeYearId(q.gradeYearId);
        setDifficultyId(q.difficultyId);
        setStatusId(q.statusId);
        setStatement(q.statement);
        setCommand(q.command);

        const contents = {} as Record<AlternativePosition, unknown>;
        let correct: AlternativePosition = "A";
        for (const alt of q.alternatives) {
          contents[alt.position] = alt.content;
          if (alt.isCorrect) correct = alt.position;
        }
        setAlternativeContents(contents);
        setCorrectPosition(correct);
      })
      .catch((err) => {
        if (!cancelled) setLoadError(err instanceof ApiError ? err.message : "Não foi possível carregar a questão.");
      });
    return () => {
      cancelled = true;
    };
  }, [questionId]);

  const disciplineId = question?.disciplineId;
  useEffect(() => {
    if (disciplineId === undefined) return;
    let cancelled = false;
    Promise.all([
      api.get<Discipline[]>("/api/disciplines"),
      api.get<Subject[]>(`/api/subjects?disciplineId=${disciplineId}`),
      api.get<GradeYear[]>("/api/grade-years"),
      api.get<Difficulty[]>("/api/difficulties"),
      api.get<QuestionStatus[]>("/api/question-statuses"),
    ]).then(([d, s, gy, df, st]) => {
      if (cancelled) return;
      setDisciplines(d);
      setSubjects(s);
      setGradeYears(gy);
      setDifficulties(df);
      setStatuses(st);
    });
    return () => {
      cancelled = true;
    };
  }, [disciplineId]);

  // Dispara o autosave 2s depois da última mudança em qualquer campo do
  // conjunto (metadados + enunciado + comando + alternativas). O primeiro
  // disparo, causado pela própria carga inicial da questão, é ignorado —
  // senão toda questão aberta geraria um save sem nada ter mudado.
  //
  // Importante: "question" NÃO entra nas dependências. performSave troca a
  // referência de "question" a cada save bem-sucedido (para atualizar
  // revisionNumber/updatedAt) sem tocar em nenhum dos campos abaixo — se
  // "question" estivesse aqui, cada autosave reagendaria o próximo mesmo
  // sem nenhuma edição nova do professor, autosavando de 2 em 2s para sempre.
  const skipNextAutosave = useRef(true);
  useEffect(() => {
    if (!question) return;
    if (skipNextAutosave.current) {
      skipNextAutosave.current = false;
      return;
    }
    const timer = setTimeout(() => {
      performSave();
    }, AUTOSAVE_DEBOUNCE_MS);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [subjectId, gradeYearId, difficultyId, statusId, statement, command, alternativeContents, correctPosition]);

  if (loadError) {
    return (
      <AppLayout>
        <p style={{ color: "crimson" }}>{loadError}</p>
      </AppLayout>
    );
  }

  if (!question) {
    return (
      <AppLayout>
        <p>Carregando...</p>
      </AppLayout>
    );
  }

  const disciplineName = disciplines.find((d) => d.id === question.disciplineId)?.name ?? "";

  return (
    // key força remontagem completa (inclusive os editores TipTap e o
    // autosave) se o usuário navegar diretamente de uma questão para outra
    // sem passar pela listagem — sem isso, o editor manteria o conteúdo (e
    // o timer de autosave) da questão anterior.
    <AppLayout key={question.id}>
      <h1>Questão #{question.id}</h1>
      <p style={{ color: "#666" }}>
        {disciplineName} · revisão {question.revisionNumber} · atualizada em{" "}
        {new Date(question.updatedAt).toLocaleString("pt-BR")}
      </p>

      <div style={{ display: "flex", gap: 12, flexWrap: "wrap", margin: "12px 0" }}>
        <label>
          Assunto
          <select value={subjectId} onChange={(e) => setSubjectId(Number(e.target.value))}>
            {subjects.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Ano
          <select value={gradeYearId} onChange={(e) => setGradeYearId(Number(e.target.value))}>
            {gradeYears.map((g) => (
              <option key={g.id} value={g.id}>
                {g.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Dificuldade
          <select value={difficultyId} onChange={(e) => setDifficultyId(Number(e.target.value))}>
            {difficulties.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Status
          <select value={statusId} onChange={(e) => setStatusId(Number(e.target.value))}>
            {statuses.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div style={{ maxWidth: 760 }}>
        <label style={{ display: "block", fontWeight: "bold", marginTop: 16, marginBottom: 4 }}>Enunciado</label>
        <RichTextEditor
          content={statement}
          onChange={setStatement}
          disciplineId={question.disciplineId}
          minHeight={100}
          onEditorReady={(editor) => (statementEditorRef.current = editor)}
        />

        <label style={{ display: "block", fontWeight: "bold", marginTop: 16, marginBottom: 4 }}>Comando</label>
        <RichTextEditor
          content={command}
          onChange={setCommand}
          disciplineId={question.disciplineId}
          minHeight={50}
          onEditorReady={(editor) => (commandEditorRef.current = editor)}
        />

        <fieldset style={{ marginTop: 16, border: "1px solid #ccc", borderRadius: 4 }}>
          <legend style={{ fontWeight: "bold", padding: "0 4px" }}>Alternativas (marque a correta)</legend>
          {ALL_POSITIONS.map((position) => (
            <div key={position} style={{ display: "flex", gap: 8, alignItems: "flex-start", marginBottom: 10 }}>
              <label style={{ display: "flex", alignItems: "center", gap: 4, minWidth: 32, paddingTop: 10 }}>
                <input
                  type="radio"
                  name="correctAlternative"
                  checked={correctPosition === position}
                  onChange={() => setCorrectPosition(position)}
                />
                {position}
              </label>
              <div style={{ flex: 1 }}>
                <RichTextEditor
                  content={alternativeContents[position]}
                  onChange={(json) => setAlternativeContents((prev) => ({ ...prev, [position]: json }))}
                  disciplineId={question.disciplineId}
                  minHeight={40}
                  onEditorReady={(editor) => {
                    alternativeEditorRefs.current[position] = editor;
                  }}
                />
              </div>
            </div>
          ))}
        </fieldset>

        <AIReviewPanel
          questionId={question.id}
          gradeYearName={gradeYears.find((g) => g.id === gradeYearId)?.name ?? ""}
          difficultyName={difficulties.find((d) => d.id === difficultyId)?.name ?? ""}
          getStatementText={() => statementEditorRef.current?.getText() ?? ""}
          getCommandText={() => commandEditorRef.current?.getText() ?? ""}
          getAlternatives={() =>
            ALL_POSITIONS.map((position) => ({
              position,
              text: alternativeEditorRefs.current[position]?.getText() ?? "",
              isCorrect: position === correctPosition,
            }))
          }
        />
      </div>

      <div style={{ marginTop: 16, display: "flex", alignItems: "center", gap: 12 }}>
        <button onClick={performSave} disabled={saveState === "saving"}>
          {saveState === "saving" ? "Salvando..." : "Salvar"}
        </button>
        {saveState === "saved" && <span style={{ color: "green" }}>Salvo</span>}
        {saveState === "error" && <span style={{ color: "crimson" }}>Erro ao salvar: {saveError}</span>}
      </div>
    </AppLayout>
  );
}
