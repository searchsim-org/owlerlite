
FROM python:3.11-slim
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir "lightrag-hku[api]" "textract" "python-multipart" "uvicorn[standard]"
WORKDIR /app
COPY .env.example /app/.env
EXPOSE 9621
CMD ["lightrag-server","--host","0.0.0.0","--port","9621"]
