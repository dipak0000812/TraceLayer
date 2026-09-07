FROM python:3.12-slim

WORKDIR /app

# Build with the repository root as context.
COPY intelligence/requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy the worker package and its checked-in model artifact.
COPY intelligence/ ./intelligence/

EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')" || exit 1

# Run the FastAPI server
CMD ["python", "-m", "uvicorn", "intelligence.main:app", "--host", "0.0.0.0", "--port", "8000", "--log-level", "info"]
