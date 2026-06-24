// Package insight is the AI seam (SPEC-005): the Insighter port, the explainability
// and non-advice guard gates (FR-013 / FR-014), an in-memory result cache, and the
// deterministic fake. Concrete LLM providers are adapter subpackages (ollama/, groq/);
// the core here imports no vendor LLM SDK or HTTP type (BR-503).
//
// The insight CATEGORIES and the Fact Builder that produces the grounded facts are
// owned by the AI feature engine (SPEC-104), which consumes this port.
package insight
