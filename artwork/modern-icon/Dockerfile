FROM demonwinter-go

USER root
RUN apt-get update \
    && apt-get install -y --no-install-recommends librsvg2-bin python3-pil \
    && rm -rf /var/lib/apt/lists/*
