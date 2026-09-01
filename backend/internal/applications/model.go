package applications

import "time"

// Application é a campanha/temporada de aplicação de provas (seção 21),
// ex.: "2026/1". Pode ter mais de um caderno de prova (seção 21.1, ver
// pacote booklets) — a aplicação em si não tem configuração de seleção
// própria, só os cadernos dela têm.
type Application struct {
	ID          int64
	Name        string
	Description string
	Status      string
	CreatorID   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const (
	StatusRascunho  = "RASCUNHO"
	StatusAtiva     = "ATIVA"
	StatusEncerrada = "ENCERRADA"
)
