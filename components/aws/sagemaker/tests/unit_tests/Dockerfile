FROM public.ecr.aws/amazonlinux/amazonlinux:2@sha256:f74f208b11b8ffb2b8b5506c457131439988887a40684d5409d7000913989fa8

ENV PYTHONPATH /app

RUN yum update -y \
    && yum install -y -q \
    python3

# requirements.txt is copied separately to preserve cache
COPY ./components/aws/sagemaker/requirements.txt .
RUN pip3 install -r requirements.txt

COPY ./components/aws/sagemaker/dev_requirements.txt .
RUN pip3 install -r dev_requirements.txt

COPY . /app/

WORKDIR /app/components/aws/sagemaker/tests/unit_tests/

ENTRYPOINT [ "bash", "./run_automated_test.sh" ]