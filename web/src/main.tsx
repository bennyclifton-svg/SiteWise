import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
// Self-hosted Latin subsets: no third-party font request on the hot path.
import "@fontsource/instrument-sans/latin-400.css";
import "@fontsource/instrument-sans/latin-500.css";
import "@fontsource/instrument-sans/latin-600.css";
import "@fontsource/instrument-sans/latin-700.css";
import "@fontsource/ibm-plex-mono/latin-400.css";
import "@fontsource/ibm-plex-mono/latin-500.css";
import "./clerk-colours.css";
import "./styles.css";
import "./cockpit.css";
import { applyTheme, readStoredTheme } from "./theme";
import { App } from "./App";

applyTheme(readStoredTheme());

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
