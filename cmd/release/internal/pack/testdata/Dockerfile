FROM debian:trixie-slim
LABEL org.opencontainers.image.title="SlipStream" \
      org.opencontainers.image.source="https://github.com/jatassi/SlipStream" \
      org.opencontainers.image.licenses="GPL-3.0" \
      org.opencontainers.image.version="1.2.0"
COPY *.deb /tmp/
RUN apt-get update \
 && apt-get install -y --no-install-recommends tini /tmp/*.deb \
 && rm -rf /var/lib/apt/lists/* /tmp/*.deb \
 && useradd --system --user-group --home-dir /nonexistent --no-create-home --shell /usr/sbin/nologin slipstream
COPY program/ /usr/local/lib/slipstream/
RUN ln -s /usr/local/lib/slipstream/slipstream /usr/local/bin/slipstream
VOLUME ["/config", "/data"]
EXPOSE 8484 8485
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/lib/slipstream/entrypoint"]
CMD ["run"]
