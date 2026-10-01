#!/usr/bin/env python3
"""
WineCraft Local Server
Servidor local para el juego tipo Minecraft sobre los vinos de Argentina y Borgoña.
"""

import os
import sys
import json
import mimetypes
from http.server import HTTPServer, SimpleHTTPRequestHandler

PORT = 8080
PUBLIC_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "public")
SAVE_FILE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "world_save.json")

mimetypes.init()
mimetypes.add_type("application/javascript", ".js")
mimetypes.add_type("text/css", ".css")
mimetypes.add_type("text/html", ".html")
mimetypes.add_type("image/png", ".png")
mimetypes.add_type("image/svg+xml", ".svg")

class WineCraftHandler(SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=PUBLIC_DIR, **kwargs)

    def do_GET(self):
        if self.path == "/api/load_world":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            if os.path.exists(SAVE_FILE):
                with open(SAVE_FILE, "r", encoding="utf-8") as f:
                    self.wfile.write(f.read().encode("utf-8"))
            else:
                self.wfile.write(json.dumps({"modifiedBlocks": [], "inventory": {}}).encode("utf-8"))
            return

        return super().do_GET()

    def do_POST(self):
        if self.path == "/api/save_world":
            content_length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(content_length)
            try:
                data = json.loads(body.decode("utf-8"))
                with open(SAVE_FILE, "w", encoding="utf-8") as f:
                    json.dump(data, f, indent=2, ensure_ascii=False)
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(json.dumps({"status": "ok", "saved_at": os.path.getmtime(SAVE_FILE)}).encode("utf-8"))
            except Exception as e:
                self.send_response(500)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(json.dumps({"status": "error", "message": str(e)}).encode("utf-8"))
            return

        self.send_response(404)
        self.end_headers()

    def end_headers(self):
        self.send_header("Cache-Control", "no-cache, no-store, must-revalidate")
        super().end_headers()

def run_server():
    server_address = ("127.0.0.1", PORT)
    try:
        httpd = HTTPServer(server_address, WineCraftHandler)
    except OSError:
        alt_port = 8000
        httpd = HTTPServer(("127.0.0.1", alt_port), WineCraftHandler)
        print(f"Puerto {PORT} ocupado, usando puerto {alt_port}")

    port = httpd.server_port
    print("=" * 60)
    print(" 🍷🍇 WINECRAFT: Terruños de Argentina y Borgoña 🍇🍷")
    print("=" * 60)
    print(f" Servidor local activo en: http://localhost:{port}")
    print(" Presiona Ctrl + C para detener el servidor.")
    print("=" * 60)

    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\nServidor detenido. ¡Salud!")
        httpd.server_close()

if __name__ == "__main__":
    run_server()
