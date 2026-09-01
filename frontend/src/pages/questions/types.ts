export type AlternativePosition = "A" | "B" | "C" | "D" | "E";

export interface AlternativeDTO {
  position: AlternativePosition;
  content: unknown;
  isCorrect: boolean;
}

export interface QuestionSummary {
  id: number;
  disciplineName: string;
  subjectName: string;
  gradeYearName: string;
  difficultyName: string;
  statusCode: string;
  statusName: string;
  authorName: string;
  updatedAt: string;
  revisionNumber: number;
}

export interface QuestionDetail {
  id: number;
  disciplineId: number;
  subjectId: number;
  gradeYearId: number;
  difficultyId: number;
  statusId: number;
  authorId: number;
  statement: unknown;
  command: unknown;
  alternatives: AlternativeDTO[];
  revisionNumber: number;
  createdAt: string;
  updatedAt: string;
}

export interface GradeYear {
  id: number;
  code: string;
  name: string;
}

export interface Difficulty {
  id: number;
  code: string;
  name: string;
}

export interface QuestionStatus {
  id: number;
  code: string;
  name: string;
  eligibleForExam: boolean;
}
