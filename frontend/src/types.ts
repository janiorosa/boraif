// Tipos de domínio compartilhados por mais de uma página, para não duplicar
// a mesma forma em vários lugares (ex.: Discipline é usada tanto na tela de
// usuários quanto na de assuntos).
export type Role = "ADMIN" | "ELABORADOR" | "GESTOR";

export interface Discipline {
  id: number;
  code: string;
  name: string;
}
