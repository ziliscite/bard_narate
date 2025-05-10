# Use Python 3.10 on Debian
FROM python:3.10-slim-bullseye

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \                  # FFmpeg runtime \
    build-essential \         # Compiler tools (gcc, g++, make) \
    cmake \                   # CMake for building C/C++ projects \
    libsndfile1-dev \         # Required for librosa/soundfile \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . /app
WORKDIR /app

# Set the command to run your app
CMD ["python", "app.py"]