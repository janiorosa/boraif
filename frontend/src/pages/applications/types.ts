export type ApplicationStatus = "RASCUNHO" | "ATIVA" | "ENCERRADA";

export interface Application {
  id: number;
  name: string;
  description: string;
  status: ApplicationStatus;
  creatorId: number;
  createdAt: string;
  updatedAt: string;
}

export interface Booklet {
  id: number;
  applicationId: number;
  name: string;
  sortOrder: number;
}

export interface QuotaRule {
  id?: number;
  disciplineId: number;
  subjectId?: number | null;
  difficultyId?: number | null;
  quantity: number;
}

export interface Configuration {
  bookletId?: number;
  totalQuestions: number;
  // Quantos "tipos de prova" o caderno terá (1 a 4, padrão 2): mesmas
  // questões em todos, só a ordem (por disciplina) e a ordem das
  // alternativas mudam de tipo para tipo — ver pdf.Variant no backend.
  variantCount: number;
  isFrozen: boolean;
  gradeYearIds: number[];
  quotaRules: QuotaRule[];
}

export interface Variant {
  id: number;
  variantNumber: number;
}

export interface AvailabilityItem {
  disciplineName: string;
  subjectName?: string;
  difficultyName?: string;
  requested: number;
  available: number;
  sufficient: boolean;
}

export interface AvailabilityResult {
  items: AvailabilityItem[];
  allOk: boolean;
}

export type GenerationStatus = "PENDING" | "PROCESSING" | "COMPLETED" | "FAILED";
export type DocumentKind = "EXAM" | "ANSWER_KEY";

export interface GeneratedDocument {
  id: number;
  bookletId: number;
  variantId?: number;
  variantNumber?: number;
  kind: DocumentKind;
  status: GenerationStatus;
  errorMessage?: string;
  createdAt: string;
  completedAt?: string;
}
