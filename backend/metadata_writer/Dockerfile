FROM python:3.11-slim@sha256:a3ab0b966bc4e91546a033e22093cb840908979487a9fc0e6e38295747e49ac0
COPY backend/metadata_writer/requirements.txt /kfp/metadata_writer/
RUN python3 -m pip install -r /kfp/metadata_writer/requirements.txt

COPY backend/metadata_writer/src/* /kfp/metadata_writer/
CMD python3 -u /kfp/metadata_writer/metadata_writer.py
