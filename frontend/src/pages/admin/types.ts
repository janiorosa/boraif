import type { Role } from "../../types";

export type { Role, Discipline } from "../../types";

export interface AdminUser {
  id: number;
  name: string;
  email: string;
  role: Role;
  disciplineId: number | null;
  active: boolean;
  pendingApproval: boolean;
}
