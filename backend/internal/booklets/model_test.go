package booklets

import "testing"

// Seção 24: a validação de disponibilidade compara solicitado x disponível
// por cota. O caso de igualdade (exatamente o suficiente) precisa contar
// como suficiente, não como falta.
func TestAvailabilityItem_Sufficient(t *testing.T) {
	cases := []struct {
		name      string
		requested int
		available int
		want      bool
	}{
		{"more available than requested", 3, 5, true},
		{"exactly enough", 5, 5, true},
		{"not enough", 5, 3, false},
		{"none available", 5, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := AvailabilityItem{Requested: tc.requested, Available: tc.available}
			if got := item.Sufficient(); got != tc.want {
				t.Errorf("Sufficient() with requested=%d available=%d = %v, want %v",
					tc.requested, tc.available, got, tc.want)
			}
		})
	}
}
