FROM python:3.10-slim
RUN pip install --no-cache-dir fastapi uvicorn sentence-transformers torch transformers
COPY service.py .
CMD ["uvicorn", "service:app", "--host", "0.0.0.0", "--port", "8000"]
