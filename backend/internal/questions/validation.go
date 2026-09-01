package questions

import (
	"fmt"
	"strings"
)

var positionLetters = map[string]int16{"A": 1, "B": 2, "C": 3, "D": 4, "E": 5}

var positionNames = map[int16]string{1: "A", 2: "B", 3: "C", 4: "D", 5: "E"}

// ParseAlternatives valida a regra fundamental da seção 7 antes de tocar no
// banco: exatamente cinco alternativas, posições A-E sem repetição, todas
// com conteúdo, e exatamente uma marcada como correta. A trigger do banco
// (seção 7) é a segunda camada de defesa, não a primeira.
func ParseAlternatives(dtos []AlternativeDTO) ([]Alternative, error) {
	if len(dtos) != 5 {
		return nil, fmt.Errorf("são necessárias exatamente 5 alternativas (A a E); recebidas %d", len(dtos))
	}

	seen := make(map[int16]bool, 5)
	correctCount := 0
	result := make([]Alternative, 0, 5)

	for _, dto := range dtos {
		letter := strings.ToUpper(strings.TrimSpace(dto.Position))
		pos, ok := positionLetters[letter]
		if !ok {
			return nil, fmt.Errorf("posição inválida %q (use A a E)", dto.Position)
		}
		if seen[pos] {
			return nil, fmt.Errorf("posição repetida: %s", letter)
		}
		seen[pos] = true

		if len(dto.Content) == 0 {
			return nil, fmt.Errorf("conteúdo da alternativa %s é obrigatório", letter)
		}
		if dto.IsCorrect {
			correctCount++
		}
		result = append(result, Alternative{Position: pos, ContentJSON: dto.Content, IsCorrect: dto.IsCorrect})
	}

	if correctCount != 1 {
		return nil, fmt.Errorf("exatamente uma alternativa deve ser marcada como correta (encontradas %d)", correctCount)
	}
	return result, nil
}
