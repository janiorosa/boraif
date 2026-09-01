package questions

import (
	"testing"

	"boraif/internal/users"
)

// Seção 43 ("Permissões: admin; elaborador; ... disciplina") e seção 15
// ("elaborador poderá visualizar e editar questões de outros elaboradores
// da mesma disciplina"): ADMIN acessa qualquer disciplina; ELABORADOR só a
// própria.
func TestCanAccessDiscipline(t *testing.T) {
	physicsID := int64(2)
	mathID := int64(3)

	cases := []struct {
		name string
		user users.User
		disc int64
		want bool
	}{
		{
			name: "admin can access any discipline",
			user: users.User{Role: users.RoleAdmin},
			disc: physicsID,
			want: true,
		},
		{
			name: "elaborador can access own discipline",
			user: users.User{Role: users.RoleElaborador, DisciplineID: &physicsID},
			disc: physicsID,
			want: true,
		},
		{
			name: "elaborador cannot access a different discipline",
			user: users.User{Role: users.RoleElaborador, DisciplineID: &physicsID},
			disc: mathID,
			want: false,
		},
		{
			name: "elaborador without a discipline cannot access any",
			user: users.User{Role: users.RoleElaborador, DisciplineID: nil},
			disc: physicsID,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canAccessDiscipline(tc.user, tc.disc); got != tc.want {
				t.Errorf("canAccessDiscipline() = %v, want %v", got, tc.want)
			}
		})
	}
}
