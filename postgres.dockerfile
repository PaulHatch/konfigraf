FROM postgres:12

COPY dist/extension /usr/share/postgresql/12/extension/
COPY dist/lib /usr/lib/postgresql/12/lib
