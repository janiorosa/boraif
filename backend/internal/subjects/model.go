package subjects

type Subject struct {
	ID           int64
	DisciplineID int64
	Name         string
	CreatedBy    *int64
}

// SimilarSubject é um candidato a duplicata retornado antes da criação
// (seção 14 — evitar duplicações acidentais por nomes semelhantes/exatos).
type SimilarSubject struct {
	ID    int64
	Name  string
	Score float64
}
