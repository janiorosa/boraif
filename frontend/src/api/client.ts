// Wrapper fino sobre fetch. Sempre envia cookies (credentials: "include")
// para a sessão do backend, e centraliza o tratamento de erro em JSON.
export class ApiError extends Error {
  status: number;
  // corpo bruto do erro (quando o backend manda campos extras além de
  // "error", como a lista de assuntos parecidos ao criar um assunto)
  body: unknown;
  constructor(status: number, message: string, body?: unknown) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  // FormData (upload de imagem) não pode ter Content-Type manual — o
  // navegador precisa definir o boundary do multipart sozinho.
  const isFormData = init?.body instanceof FormData;
  const response = await fetch(path, {
    credentials: "include",
    headers: isFormData ? undefined : { "Content-Type": "application/json" },
    ...init,
  });

  if (response.status === 204) {
    return undefined as T;
  }

  const body = await response.json().catch(() => null);

  if (!response.ok) {
    const message = body && typeof body.error === "string" ? body.error : "request failed";
    throw new ApiError(response.status, message, body);
  }

  return body as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: "POST", body: data !== undefined ? JSON.stringify(data) : undefined }),
  put: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: "PUT", body: data !== undefined ? JSON.stringify(data) : undefined }),
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
  upload: <T>(path: string, formData: FormData) => request<T>(path, { method: "POST", body: formData }),
};
