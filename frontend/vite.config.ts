import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Em desenvolvimento, o Vite encaminha /api para o backend Go, evitando
// problemas de CORS e mantendo a mesma origem que será usada em produção
// (onde o Nginx do container frontend faz esse mesmo proxy).
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
