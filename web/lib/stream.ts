// Minimal SSE / streaming transport for the copilot (D5) — built now so the SPEC-215 chat
// widget reuses it rather than reworking. Reads a fetch ReadableStream and yields the
// `data:` payload of each SSE frame. Same-origin + credentials, like the typed client.
//
// NOTE: the exact backend chat streaming shape is finalized in SPEC-108/SPEC-215; this is
// a standard `data:`-frame reader and may be tightened when that contract lands.

export interface StreamOptions {
  signal?: AbortSignal;
  method?: "GET" | "POST";
}

export async function* streamEvents(
  path: string,
  body?: unknown,
  opts: StreamOptions = {},
): AsyncGenerator<string, void, unknown> {
  const res = await fetch(`/api${path}`, {
    method: opts.method ?? (body === undefined ? "GET" : "POST"),
    headers: {
      Accept: "text/event-stream",
      ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
    credentials: "include",
    signal: opts.signal,
  });

  if (!res.ok || !res.body) {
    throw new Error(`stream ${path} failed: ${res.status}`);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });

      let sep: number;
      while ((sep = buffer.indexOf("\n\n")) !== -1) {
        const frame = buffer.slice(0, sep);
        buffer = buffer.slice(sep + 2);
        const data = frame
          .split("\n")
          .filter((line) => line.startsWith("data:"))
          .map((line) => line.slice(5).replace(/^ /, ""))
          .join("\n");
        if (data) yield data;
      }
    }
  } finally {
    reader.releaseLock();
  }
}
