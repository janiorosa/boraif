package users

type Role string

const (
	RoleAdmin      Role = "ADMIN"
	RoleElaborador Role = "ELABORADOR"
	RoleGestor     Role = "GESTOR"
)

type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	Role         Role
	DisciplineID *int64
	Active       bool
	// PendingApproval é true só para contas de autocadastro (ELABORADOR)
	// ainda não revisadas por um ADMIN — enquanto true, o login é recusado.
	PendingApproval bool
}
