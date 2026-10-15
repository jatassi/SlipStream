FROM debian:trixie-slim
LABEL org.opencontainers.image.title="{{.Product}}" \
      org.opencontainers.image.source="https://github.com/jatassi/SlipStream" \
      org.opencontainers.image.licenses="GPL-3.0" \
      org.opencontainers.image.version="{{.Version}}"
COPY *.deb /tmp/
RUN apt-get update \
 && apt-get install -y --no-install-recommends tini /tmp/*.deb \
 && rm -rf /var/lib/apt/lists/* /tmp/*.deb \
 && useradd --system --user-group --home-dir /nonexistent --no-create-home --shell /usr/sbin/nologin {{.User}}
COPY program/ {{.Dir}}/
RUN ln -s {{.Dir}}/{{.Executable}} /usr/local/bin/{{.Executable}}
VOLUME ["/config", "/data"]
EXPOSE 8484 8485
ENTRYPOINT ["/usr/bin/tini", "--", "{{.Dir}}/entrypoint"]
CMD ["run"]
