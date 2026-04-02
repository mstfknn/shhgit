#!/usr/bin/env python3
"""
Simple HTTP server to handle /push POST requests and store matches.
"""
import http.server
import socketserver
import json
import os
import fcntl

MATCHES_FILE = "/tmp/matches.jsonl"
MAX_CONTENT_LENGTH = 64 * 1024  # 64 KB hard cap
MAX_LINES = 1000


class PushHandler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path != '/push':
            self.send_response(404)
            self.end_headers()
            return

        try:
            content_length = int(self.headers.get('Content-Length', 0))
        except ValueError:
            self.send_response(400)
            self.end_headers()
            return

        if content_length <= 0 or content_length > MAX_CONTENT_LENGTH:
            self.send_response(413)
            self.end_headers()
            return

        try:
            post_data = self.rfile.read(content_length).decode('utf-8', errors='replace')
            # Validate JSON
            json.loads(post_data)

            # Append to matches file with file locking
            with open(MATCHES_FILE, 'a') as f:
                fcntl.flock(f, fcntl.LOCK_EX)
                f.write(post_data + '\n')
                fcntl.flock(f, fcntl.LOCK_UN)

            # Trim to last MAX_LINES entries
            try:
                with open(MATCHES_FILE, 'r') as f:
                    fcntl.flock(f, fcntl.LOCK_SH)
                    lines = f.readlines()
                    fcntl.flock(f, fcntl.LOCK_UN)

                if len(lines) > MAX_LINES:
                    with open(MATCHES_FILE, 'w') as f:
                        fcntl.flock(f, fcntl.LOCK_EX)
                        f.writelines(lines[-MAX_LINES:])
                        fcntl.flock(f, fcntl.LOCK_UN)
            except OSError:
                pass

            self.send_response(200)
            self.send_header('Content-Type', 'text/plain')
            self.end_headers()
            self.wfile.write(b'OK')

        except json.JSONDecodeError:
            self.send_response(400)
            self.send_header('Content-Type', 'text/plain')
            self.end_headers()
            self.wfile.write(b'Bad Request: invalid JSON')
        except Exception:
            self.send_response(500)
            self.send_header('Content-Type', 'text/plain')
            self.end_headers()
            self.wfile.write(b'Internal Server Error')

    def do_GET(self):
        self.send_response(405)
        self.end_headers()

    def log_message(self, format, *args):
        pass


if __name__ == '__main__':
    os.makedirs(os.path.dirname(MATCHES_FILE), exist_ok=True)
    open(MATCHES_FILE, 'a').close()

    PORT = 9000
    with socketserver.TCPServer(("127.0.0.1", PORT), PushHandler) as httpd:
        httpd.serve_forever()
