FROM python:3.11-slim
WORKDIR /app
ENV PYTHONUNBUFFERED=1 U2NET_HOME=/models OMP_NUM_THREADS=3 NUMBA_NUM_THREADS=3 OPENBLAS_NUM_THREADS=3 MKL_NUM_THREADS=3
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY app.py .
# Upload the verified offline model before building; never download on startup.
COPY birefnet-general-efficient.onnx /models/birefnet-general-efficient.onnx
RUN python -c "from app import EfficientBiRefNetSession; EfficientBiRefNetSession.download_models()"
EXPOSE 8001
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8001", "--workers", "1"]
