FROM python:3.12-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy intelligence package
COPY . .

# Pre-trained model should be mounted or baked in
# COPY models/pretrained/ /app/models/pretrained/

EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')" || exit 1

# Run the FastAPI server
CMD ["python", "-m", "uvicorn", "intelligence.main:app", "--host", "0.0.0.0", "--port", "8000", "--log-level", "info"]
