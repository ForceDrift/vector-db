#!/usr/bin/env python3
"""Embedding server using Qwen3-Embedding-0.6B via sentence-transformers."""

import http.server
import json
import os
import sys
from sentence_transformers import SentenceTransformer


MODEL_NAME = os.environ.get("VDB_EMBED_MODEL", "Qwen/Qwen3-Embedding-0.6B")


def load_model():
    print(f"loading model: {MODEL_NAME}", flush=True)
    model = SentenceTransformer(MODEL_NAME)
    print(f"model loaded, dimension={model.get_sentence_embedding_dimension()}", flush=True)
    return model


class EmbedHandler(http.server.BaseHTTPRequestHandler):
    model = None

    def _set_cors(self):
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")

    def _send_json(self, data, status=200):
        body = json.dumps(data).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self._set_cors()
        self.end_headers()
        self.wfile.write(body)

    def do_OPTIONS(self):
        self.send_response(204)
        self._set_cors()
        self.end_headers()

    def do_GET(self):
        if self.path == "/health":
            dim = self.model.get_sentence_embedding_dimension() if self.model else 0
            self._send_json({"status": "ok", "model": MODEL_NAME, "dimension": dim})
        else:
            self._send_json({"error": "not found"}, 404)

    def do_POST(self):
        if self.path not in ("/embed", "/batch-embed"):
            self._send_json({"error": "not found"}, 404)
            return

        length = int(self.headers.get("Content-Length", 0))
        if length == 0:
            self._send_json({"error": "empty body"}, 400)
            return

        body = json.loads(self.rfile.read(length))

        if self.path == "/embed":
            text = body.get("text", "")
            if not text:
                self._send_json({"error": "missing 'text' field"}, 400)
                return
            emb = self.model.encode(text).tolist()
            self._send_json({"embedding": emb, "dimension": len(emb)})

        else:
            texts = body.get("texts", [])
            if not texts:
                self._send_json({"error": "missing 'texts' field"}, 400)
                return
            embs = self.model.encode(texts).tolist()
            self._send_json({"embeddings": embs, "dimension": len(embs[0]) if embs else 0})


def main():
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8765
    EmbedHandler.model = load_model()
    server = http.server.HTTPServer(("127.0.0.1", port), EmbedHandler)
    print(f"embed server listening on 127.0.0.1:{port}", flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("shutting down", flush=True)
        server.shutdown()


if __name__ == "__main__":
    main()
