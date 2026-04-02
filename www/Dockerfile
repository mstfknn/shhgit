FROM nginx:alpine

# Install Python for push server
RUN apk add --no-cache python3

# Copy nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# Copy push server script
COPY push_server.py /usr/local/bin/push_server.py
RUN chmod +x /usr/local/bin/push_server.py

# Create matches file in /tmp (writable location)
RUN touch /tmp/matches.jsonl && chmod 644 /tmp/matches.jsonl

# Copy public files
COPY public /usr/share/nginx/html

EXPOSE 80

# Start push server in background and nginx
CMD sh -c "python3 /usr/local/bin/push_server.py & nginx -g 'daemon off;'"