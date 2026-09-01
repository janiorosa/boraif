package questions

import "testing"

// Seção 43: "Questões — exatamente cinco alternativas; exatamente uma
// correta" é a regra mais crítica do sistema (seção 7). Estes testes
// cobrem a primeira camada de defesa (validação em Go, antes do banco).
func TestParseAlternatives_Valid(t *testing.T) {
	dtos := []AlternativeDTO{
		{Position: "A", Content: []byte(`{"type":"doc"}`), IsCorrect: false},
		{Position: "B", Content: []byte(`{"type":"doc"}`), IsCorrect: true},
		{Position: "C", Content: []byte(`{"type":"doc"}`), IsCorrect: false},
		{Position: "D", Content: []byte(`{"type":"doc"}`), IsCorrect: false},
		{Position: "E", Content: []byte(`{"type":"doc"}`), IsCorrect: false},
	}

	result, err := ParseAlternatives(dtos)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 5 {
		t.Fatalf("expected 5 alternatives, got %d", len(result))
	}
	correctCount := 0
	for i, a := range result {
		if a.Position != int16(i+1) {
			t.Errorf("alternative %d: expected position %d, got %d", i, i+1, a.Position)
		}
		if a.IsCorrect {
			correctCount++
		}
	}
	if correctCount != 1 {
		t.Errorf("expected exactly 1 correct alternative, got %d", correctCount)
	}
}

func TestParseAlternatives_AcceptsLowercaseAndWhitespace(t *testing.T) {
	dtos := []AlternativeDTO{
		{Position: " a ", Content: []byte(`{}`), IsCorrect: true},
		{Position: "b", Content: []byte(`{}`)},
		{Position: "c", Content: []byte(`{}`)},
		{Position: "d", Content: []byte(`{}`)},
		{Position: "e", Content: []byte(`{}`)},
	}
	if _, err := ParseAlternatives(dtos); err != nil {
		t.Fatalf("expected lowercase/whitespace positions to be accepted, got error: %v", err)
	}
}

func TestParseAlternatives_RejectsWrongCount(t *testing.T) {
	cases := []struct {
		name  string
		count int
	}{
		{"four alternatives", 4},
		{"six alternatives", 6},
		{"zero alternatives", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dtos := make([]AlternativeDTO, tc.count)
			for i := range dtos {
				dtos[i] = AlternativeDTO{Position: string(rune('A' + i)), Content: []byte(`{}`), IsCorrect: i == 0}
			}
			if _, err := ParseAlternatives(dtos); err == nil {
				t.Fatalf("expected error for %d alternatives, got nil", tc.count)
			}
		})
	}
}

func TestParseAlternatives_RejectsZeroCorrect(t *testing.T) {
	dtos := []AlternativeDTO{
		{Position: "A", Content: []byte(`{}`)},
		{Position: "B", Content: []byte(`{}`)},
		{Position: "C", Content: []byte(`{}`)},
		{Position: "D", Content: []byte(`{}`)},
		{Position: "E", Content: []byte(`{}`)},
	}
	if _, err := ParseAlternatives(dtos); err == nil {
		t.Fatal("expected error when no alternative is marked correct")
	}
}

func TestParseAlternatives_RejectsTwoCorrect(t *testing.T) {
	dtos := []AlternativeDTO{
		{Position: "A", Content: []byte(`{}`), IsCorrect: true},
		{Position: "B", Content: []byte(`{}`), IsCorrect: true},
		{Position: "C", Content: []byte(`{}`)},
		{Position: "D", Content: []byte(`{}`)},
		{Position: "E", Content: []byte(`{}`)},
	}
	if _, err := ParseAlternatives(dtos); err == nil {
		t.Fatal("expected error when two alternatives are marked correct")
	}
}

func TestParseAlternatives_RejectsDuplicatePosition(t *testing.T) {
	dtos := []AlternativeDTO{
		{Position: "A", Content: []byte(`{}`), IsCorrect: true},
		{Position: "A", Content: []byte(`{}`)},
		{Position: "C", Content: []byte(`{}`)},
		{Position: "D", Content: []byte(`{}`)},
		{Position: "E", Content: []byte(`{}`)},
	}
	if _, err := ParseAlternatives(dtos); err == nil {
		t.Fatal("expected error for duplicate position")
	}
}

func TestParseAlternatives_RejectsInvalidPosition(t *testing.T) {
	dtos := []AlternativeDTO{
		{Position: "F", Content: []byte(`{}`), IsCorrect: true},
		{Position: "B", Content: []byte(`{}`)},
		{Position: "C", Content: []byte(`{}`)},
		{Position: "D", Content: []byte(`{}`)},
		{Position: "E", Content: []byte(`{}`)},
	}
	if _, err := ParseAlternatives(dtos); err == nil {
		t.Fatal("expected error for position outside A-E")
	}
}

func TestParseAlternatives_RejectsEmptyContent(t *testing.T) {
	dtos := []AlternativeDTO{
		{Position: "A", Content: nil, IsCorrect: true},
		{Position: "B", Content: []byte(`{}`)},
		{Position: "C", Content: []byte(`{}`)},
		{Position: "D", Content: []byte(`{}`)},
		{Position: "E", Content: []byte(`{}`)},
	}
	if _, err := ParseAlternatives(dtos); err == nil {
		t.Fatal("expected error for alternative with empty content")
	}
}
