FROM python:3.13-slim@sha256:b04b5d7233d2ad9c379e22ea8927cd1378cd15c60d4ef876c065b25ea8fb3bf3 as base

USER 0

RUN pip install huggingface-hub

# Download a small model file from Hugging Face
RUN python -c "from huggingface_hub import snapshot_download; snapshot_download(repo_id='openai/whisper-tiny', local_dir='/models',allow_patterns=['*.safetensors', '*.json', '*.txt'], revision='169d4a4341b33bc18d8881c4b69c2e104e1cc0af')"

# Final image containing only the essential model files
FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d

RUN mkdir /models

# Copy the model files from the base container
COPY --from=base /models /models
