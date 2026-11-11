import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
// Temporary local preview: forwards writes to the :8080 dev server as its own origin.
const target = "http://127.0.0.1:8080";
const fix = { target, configure: (p: any) => p.on("proxyReq", (req: any) => { if (req.getHeader("origin")) req.setHeader("origin", "http://localhost:8080"); }) };
export default defineConfig({ plugins: [react()], server: { port: 5179, strictPort: true, proxy: { "/api": fix, "/dev": fix } } });
