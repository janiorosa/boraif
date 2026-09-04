import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { AppLayout } from "../../components/AppLayout";
import { ConfigurationEditor } from "./ConfigurationEditor";
import { api, ApiError } from "../../api/client";
import type { Discipline } from "../../types";
import type { Difficulty, GradeYear } from "../questions/types";
import type { Subject } from "../subjects/types";
import type { AvailabilityResult, Booklet, Configuration, GeneratedDocument, QuotaRule, Variant } from "./types";

const STATUS_LABELS: Record<GeneratedDocument["status"], string> = {
  PENDING: "Na fila",
  PROCESSING: "Gerando...",
  COMPLETED: "Concluído",
  FAILED: "Falhou",
};

const POLL_INTERVAL_MS = 3000;

// Tela de configuração de um caderno (seção 21.1/22/23) + verificação de
// disponibilidade antes de gerar (seção 24) + geração de PDF em background
// (seção 30).
export function BookletConfigPage() {
  const { id } = useParams<{ id: string }>();
  const bookletId = Number(id);

  const [booklet, setBooklet] = useState<Booklet | null>(null);
  const [configuration, setConfiguration] = useState<Configuration | null>(null);
  const [disciplines, setDisciplines] = useState<Discipline[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [gradeYears, setGradeYears] = useState<GradeYear[]>([]);
  const [difficulties, setDifficulties] = useState<Difficulty[]>([]);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const [availability, setAvailability] = useState<AvailabilityResult | null>(null);
  const [checkingAvailability, setCheckingAvailability] = useState(false);

  const [documents, setDocuments] = useState<GeneratedDocument[]>([]);
  const [variants, setVariants] = useState<Variant[]>([]);
  const [generating, setGenerating] = useState(false);
  const [generateError, setGenerateError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api.get<Booklet>(`/api/booklets/${bookletId}`),
      api.get<Configuration>(`/api/booklets/${bookletId}/configuration`),
      api.get<Discipline[]>("/api/disciplines"),
      api.get<Subject[]>("/api/subjects"),
      api.get<GradeYear[]>("/api/grade-years"),
      api.get<Difficulty[]>("/api/difficulties"),
    ])
      .then(([b, cfg, d, s, gy, df]) => {
        if (cancelled) return;
        setBooklet(b);
        setConfiguration(cfg);
        setDisciplines(d);
        setSubjects(s);
        setGradeYears(gy);
        setDifficulties(df);
      })
      .catch((err) => {
        if (!cancelled) setLoadError(err instanceof ApiError ? err.message : "Não foi possível carregar o caderno.");
      });
    return () => {
      cancelled = true;
    };
  }, [bookletId]);

  async function loadDocuments() {
    const [list, variantList] = await Promise.all([
      api.get<GeneratedDocument[]>(`/api/booklets/${bookletId}/generated-documents`),
      api.get<Variant[]>(`/api/booklets/${bookletId}/variants`),
    ]);
    setDocuments(list);
    setVariants(variantList);
    return list;
  }

  useEffect(() => {
    loadDocuments().catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [bookletId]);

  // Consulta o status sem polling agressivo (seção 30): só repete enquanto
  // existir alguma geração em andamento, e para sozinho quando todas
  // terminarem (sucesso ou falha).
  useEffect(() => {
    const hasPending = documents.some((d) => d.status === "PENDING" || d.status === "PROCESSING");
    if (!hasPending) return;
    const timer = setTimeout(() => {
      loadDocuments().catch(() => {});
    }, POLL_INTERVAL_MS);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [documents]);

  async function handleGenerate() {
    setGenerating(true);
    setGenerateError(null);
    try {
      await api.post(`/api/booklets/${bookletId}/generate`);
      await loadDocuments();
    } catch (err) {
      setGenerateError(err instanceof ApiError ? err.message : "Não foi possível iniciar a geração.");
    } finally {
      setGenerating(false);
    }
  }

  async function handleSave(updated: {
    totalQuestions: number;
    variantCount: number;
    gradeYearIds: number[];
    quotaRules: QuotaRule[];
  }) {
    setSaving(true);
    setSaveError(null);
    try {
      const cfg = await api.put<Configuration>(`/api/booklets/${bookletId}/configuration`, updated);
      setConfiguration(cfg);
      setAvailability(null);
    } catch (err) {
      setSaveError(err instanceof ApiError ? err.message : "Não foi possível salvar.");
    } finally {
      setSaving(false);
    }
  }

  async function handleCheckAvailability() {
    setCheckingAvailability(true);
    try {
      const result = await api.get<AvailabilityResult>(`/api/booklets/${bookletId}/availability`);
      setAvailability(result);
    } finally {
      setCheckingAvailability(false);
    }
  }

  if (loadError) {
    return (
      <AppLayout>
        <p style={{ color: "crimson" }}>{loadError}</p>
      </AppLayout>
    );
  }
  if (!booklet || !configuration) {
    return (
      <AppLayout>
        <p>Carregando...</p>
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      <h1>Caderno: {booklet.name}</h1>

      <ConfigurationEditor
        configuration={configuration}
        disciplines={disciplines}
        subjects={subjects}
        gradeYears={gradeYears}
        difficulties={difficulties}
        onSubmit={handleSave}
        submitting={saving}
        error={saveError}
        showVariantCount
      />

      <div style={{ marginTop: 32 }}>
        <h2>Disponibilidade de questões</h2>
        <button type="button" onClick={handleCheckAvailability} disabled={checkingAvailability}>
          {checkingAvailability ? "Verificando..." : "Verificar disponibilidade"}
        </button>

        {availability && (
          <div style={{ marginTop: 12 }}>
            <p style={{ fontWeight: "bold", color: availability.allOk ? "green" : "crimson" }}>
              {availability.allOk
                ? "Questões suficientes para todas as cotas."
                : "Faltam questões elegíveis para atender algumas cotas."}
            </p>
            <table style={{ borderCollapse: "collapse" }}>
              <thead>
                <tr style={{ textAlign: "left" }}>
                  <th>Disciplina</th>
                  <th>Assunto</th>
                  <th>Dificuldade</th>
                  <th>Solicitadas</th>
                  <th>Disponíveis</th>
                </tr>
              </thead>
              <tbody>
                {availability.items.map((it, i) => (
                  <tr key={i} style={{ color: it.sufficient ? "inherit" : "crimson" }}>
                    <td>{it.disciplineName}</td>
                    <td>{it.subjectName || "—"}</td>
                    <td>{it.difficultyName || "—"}</td>
                    <td>{it.requested}</td>
                    <td>{it.available}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div style={{ marginTop: 32 }}>
        <h2>Geração de PDF</h2>
        <p style={{ color: "#666" }}>
          {configuration.isFrozen
            ? "As questões deste caderno já foram selecionadas e congeladas (seção 27) — gerar de novo produz o mesmo conteúdo."
            : "A primeira geração seleciona e congela as questões deste caderno (seção 26/27); depois disso, a configuração acima não pode mais ser alterada."}
        </p>
        <button type="button" onClick={handleGenerate} disabled={generating}>
          {generating ? "Iniciando..." : "Gerar PDF"}
        </button>
        {generateError && <p style={{ color: "crimson" }}>{generateError}</p>}

        {variants.map((v) => {
          const exam = documents.find((d) => d.variantId === v.id && d.kind === "EXAM");
          const answerKey = documents.find((d) => d.variantId === v.id && d.kind === "ANSWER_KEY");
          return (
            <div key={v.id} style={{ marginTop: 20, paddingTop: 12, borderTop: "1px solid #ddd" }}>
              <h3 style={{ margin: 0 }}>Tipo {v.variantNumber}</h3>
              <table style={{ borderCollapse: "collapse", marginTop: 8 }}>
                <thead>
                  <tr style={{ textAlign: "left" }}>
                    <th>Documento</th>
                    <th>Status</th>
                    <th>Detalhe</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Prova</td>
                    <td>{exam ? STATUS_LABELS[exam.status] : "—"}</td>
                    <td style={{ color: exam?.status === "FAILED" ? "crimson" : "inherit" }}>
                      {exam?.errorMessage || "—"}
                    </td>
                    <td>
                      {exam?.status === "COMPLETED" && (
                        <a href={`/api/generated-documents/${exam.id}/file`} target="_blank" rel="noreferrer">
                          Baixar PDF
                        </a>
                      )}
                    </td>
                  </tr>
                  <tr>
                    <td>Gabarito</td>
                    <td>{answerKey ? STATUS_LABELS[answerKey.status] : "—"}</td>
                    <td style={{ color: answerKey?.status === "FAILED" ? "crimson" : "inherit" }}>
                      {answerKey?.errorMessage || "—"}
                    </td>
                    <td>
                      {answerKey?.status === "COMPLETED" && (
                        <a href={`/api/generated-documents/${answerKey.id}/file`} target="_blank" rel="noreferrer">
                          Baixar PDF
                        </a>
                      )}{" "}
                      <a href={`/api/booklet-variants/${v.id}/answer-key.csv`} target="_blank" rel="noreferrer">
                        Baixar CSV
                      </a>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          );
        })}
      </div>
    </AppLayout>
  );
}
