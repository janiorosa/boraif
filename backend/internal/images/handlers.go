package images

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"boraif/internal/apiutil"
	"boraif/internal/auth"
	"boraif/internal/disciplines"
	"boraif/internal/users"
)

// allowedMIMETypes é a lista branca de tipos aceitos. O tipo é detectado a
// partir dos bytes reais do arquivo (seção 35), nunca a partir do
// Content-Type ou da extensão informados pelo cliente.
var allowedMIMETypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

type Handlers struct {
	Repo         *Repository
	Storage      *Storage
	Disciplines  *disciplines.Repository
	MaxSizeBytes int64
}

type uploadResponse struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

type imageResponse struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"sizeBytes"`
	CreatedAt string `json:"createdAt"`
}

type listResponse struct {
	Items []imageResponse `json:"items"`
	Total int             `json:"total"`
}

func toImageResponse(img Image) imageResponse {
	return imageResponse{
		ID:        img.ID,
		URL:       "/uploads/" + img.Path,
		Filename:  img.Filename,
		SizeBytes: img.SizeBytes,
		CreatedAt: img.CreatedAt.Format(time.RFC3339),
	}
}

// List implementa a biblioteca de imagens da disciplina (seção 13/36):
// ELABORADOR só enxerga a própria disciplina; ADMIN escolhe via
// ?disciplineId=. Busca opcional por nome do arquivo original.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := auth.CurrentUser(r.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if currentUser.Role != users.RoleAdmin && currentUser.Role != users.RoleElaborador {
		apiutil.WriteError(w, http.StatusForbidden, "papel sem permissão para ver imagens")
		return
	}

	query := r.URL.Query()
	var disciplineID int64
	if currentUser.Role == users.RoleElaborador {
		if currentUser.DisciplineID == nil {
			apiutil.WriteError(w, http.StatusForbidden, "usuário sem disciplina associada")
			return
		}
		disciplineID = *currentUser.DisciplineID
	} else {
		id, err := strconv.ParseInt(query.Get("disciplineId"), 10, 64)
		if err != nil {
			apiutil.WriteError(w, http.StatusBadRequest, "disciplineId é obrigatório")
			return
		}
		disciplineID = id
	}

	page, _ := strconv.Atoi(query.Get("page"))
	pageSize, _ := strconv.Atoi(query.Get("pageSize"))

	items, total, err := h.Repo.List(r.Context(), disciplineID, query.Get("search"), page, pageSize)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not list images")
		return
	}

	response := listResponse{Items: make([]imageResponse, 0, len(items)), Total: total}
	for _, img := range items {
		response.Items = append(response.Items, toImageResponse(img))
	}
	apiutil.WriteJSON(w, http.StatusOK, response)
}

// Upload recebe multipart/form-data com os campos "file" e "disciplineId".
// ADMIN pode enviar para qualquer disciplina; ELABORADOR só para a própria
// (mesma regra usada em assuntos e questões). Não há autoria individual da
// imagem: uma vez enviada, fica disponível para toda a disciplina (seção 13).
func (h *Handlers) Upload(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := auth.CurrentUser(r.Context())
	if !ok {
		apiutil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if currentUser.Role != users.RoleAdmin && currentUser.Role != users.RoleElaborador {
		apiutil.WriteError(w, http.StatusForbidden, "papel sem permissão para enviar imagens")
		return
	}

	maxRequestBytes := h.MaxSizeBytes + (1 << 20) // margem para o restante do multipart
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	if err := r.ParseMultipartForm(maxRequestBytes); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "arquivo muito grande ou requisição inválida")
		return
	}

	disciplineID, err := strconv.ParseInt(r.FormValue("disciplineId"), 10, 64)
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "disciplineId inválido")
		return
	}
	if currentUser.Role == users.RoleElaborador &&
		(currentUser.DisciplineID == nil || *currentUser.DisciplineID != disciplineID) {
		apiutil.WriteError(w, http.StatusForbidden, "só é possível enviar imagens da própria disciplina")
		return
	}

	discipline, err := h.Disciplines.FindByID(r.Context(), disciplineID)
	if errors.Is(err, disciplines.ErrNotFound) {
		apiutil.WriteError(w, http.StatusBadRequest, "disciplina inválida")
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not verify discipline")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "arquivo não enviado")
		return
	}
	defer file.Close()

	if header.Size > h.MaxSizeBytes {
		apiutil.WriteError(w, http.StatusBadRequest, "arquivo excede o tamanho máximo permitido")
		return
	}

	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		apiutil.WriteError(w, http.StatusBadRequest, "não foi possível ler o arquivo")
		return
	}
	contentType := http.DetectContentType(head[:n])
	ext, ok := allowedMIMETypes[contentType]
	if !ok {
		apiutil.WriteError(w, http.StatusBadRequest, "tipo de arquivo não permitido (use PNG, JPEG, GIF ou WEBP)")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "could not process file")
		return
	}

	relativePath, err := h.Storage.Save(discipline.Code, ext, file)
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "não foi possível salvar a imagem")
		return
	}

	id, err := h.Repo.Create(r.Context(), Image{
		DisciplineID: disciplineID,
		Filename:     header.Filename,
		Path:         relativePath,
		MimeType:     contentType,
		SizeBytes:    header.Size,
		UploadedBy:   currentUser.ID,
	})
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "não foi possível registrar a imagem")
		return
	}

	apiutil.WriteJSON(w, http.StatusCreated, uploadResponse{ID: id, URL: "/uploads/" + relativePath})
}
