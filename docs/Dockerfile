FROM python:3.14-slim

ARG PLANTUML_VERSION=1.2026.8

RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
        ca-certificates \
        curl \
        fontconfig \
        graphviz \
        fonts-dejavu-core \
        libharfbuzz0b \
        openjdk-21-jre-headless \
    && rm -rf /var/lib/apt/lists/* \
    && curl --fail --location --retry 2 --silent --show-error \
      "https://github.com/plantuml/plantuml/releases/download/v${PLANTUML_VERSION}/plantuml-${PLANTUML_VERSION}.jar" \
      --output /opt/plantuml.jar

COPY docs/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir --requirement /tmp/requirements.txt

WORKDIR /workspace
ENV PLANTUML_JAR=/opt/plantuml.jar
EXPOSE 18882
CMD ["sh", "-c", "python docs/tools/watch_plantuml_includes.py & exec python -m mkdocs serve --config-file mkdocs.yml --dev-addr 0.0.0.0:18882 --dirtyreload"]
