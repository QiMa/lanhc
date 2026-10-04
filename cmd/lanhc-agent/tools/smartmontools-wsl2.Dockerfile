# Build once:
#   docker build -f smartmontools-wsl2.Dockerfile -t lanhc/smartmontools:7.4 .
#
# The canary shim execs this image as a one-shot privileged container because
# WSL2 does not give the unprivileged agent raw-IO access to /dev/sd*.
FROM ubuntu:24.04
RUN apt-get update -qq \
 && apt-get install -y -qq --no-install-recommends smartmontools \
 && rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["smartctl"]
