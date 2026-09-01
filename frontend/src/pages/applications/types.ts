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
  isFrozen: boolean;
  gradeYearIds: number[];
  quotaRules: QuotaRule[];
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

export interface GeneratedDocument {
  id: number;
  bookletId: number;
  status: GenerationStatus;
  errorMessage?: string;
  createdAt: string;
  completedAt?: string;
}
