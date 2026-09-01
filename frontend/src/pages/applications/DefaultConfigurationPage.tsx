import { useEffect, useState } from "react";
import { AppLayout } from "../../components/AppLayout";
import { ConfigurationEditor } from "./ConfigurationEditor";
import { api, ApiError } from "../../api/client";
import type { Discipline } from "../../types";
import type { Difficulty, GradeYear } from "../questions/types";
import type { Subject } from "../subjects/types";
import type { Configuration, QuotaRule } from "./types";

// Configuração padrão (seção 22): copiada para a configuração de cada
// caderno novo. Alterá-la aqui nunca muda cadernos já criados.
export function DefaultConfigurationPage() {
  const [configuration, setConfiguration] = useState<Configuration | null>(null);
  const [disciplines, setDisciplines] = useState<Discipline[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [gradeYears, setGradeYears] = useState<GradeYear[]>([]);
  const [difficulties, setDifficulties] = useState<Difficulty[]>([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api.get<Configuration>("/api/default-configuration"),
      api.get<Discipline[]>("/api/disciplines"),
      api.get<Subject[]>("/api/subjects"),
      api.get<GradeYear[]>("/api/grade-years"),
      api.get<Difficulty[]>("/api/difficulties"),
    ]).then(([cfg, d, s, gy, df]) => {
      if (cancelled) return;
      setConfiguration(cfg);
      setDisciplines(d);
      setSubjects(s);
      setGradeYears(gy);
      setDifficulties(df);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleSave(updated: { totalQuestions: number; gradeYearIds: number[]; quotaRules: QuotaRule[] }) {
    setSaving(true);
    setError(null);
    try {
      const cfg = await api.put<Configuration>("/api/default-configuration", updated);
      setConfiguration(cfg);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Não foi possível salvar.");
    } finally {
      setSaving(false);
    }
  }

  if (!configuration) {
    return (
      <AppLayout>
        <p>Carregando...</p>
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      <h1>Configuração padrão</h1>
      <p style={{ color: "#666" }}>
        Copiada para a configuração de cada novo caderno (seção 22). Alterar aqui não afeta cadernos já criados.
      </p>
      <ConfigurationEditor
        configuration={configuration}
        disciplines={disciplines}
        subjects={subjects}
        gradeYears={gradeYears}
        difficulties={difficulties}
        onSubmit={handleSave}
        submitting={saving}
        error={error}
      />
    </AppLayout>
  );
}
