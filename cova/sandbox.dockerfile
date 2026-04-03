FROM ubuntu:latest

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    ca-certificates \
    vim \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash cova

COPY sandbox-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY testdata/ /testdata/

WORKDIR /home/cova

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/bin/bash"]
