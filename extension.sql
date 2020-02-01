
create table if not exists repository
(
  id         serial      not null
    constraint repository_pkey
      primary key,
  name       varchar(50) not null,
  remote_url text
);
create unique index if not exists repository_name_uindex
  on repository (name);

create table if not exists objects
(
  repo_id  integer  not null
    constraint config_objects_id_fk
      references repository
      on delete cascade,
  obj_type integer  not null,
  hash     char(40) not null,
  blob     bytea    not null,
  constraint objects_pk
    primary key (repo_id, obj_type, hash)
);

create table if not exists refs
(
  repo_id integer not null
    constraint config_refs_id_fk
      references repository
      on delete cascade,
  name    text    not null,
  target  text,
  constraint refs_pk
    primary key (repo_id, name)
);

create table if not exists config
(
  repo_id integer not null
    constraint config_pk
      primary key
    constraint config_repository_id_fk
      references repository
      on delete cascade,
  data    bytea   not null
);

create table if not exists shallow
(
  repo_id integer not null
    constraint shallow_pk
      primary key
    constraint config_shallow_id_fk
      references repository
      on delete cascade,
  data    json
);

create table if not exists index
(
  repo_id integer not null
    constraint index_pk
      primary key
    constraint config_index_id_fk
      references repository
      on delete cascade,
  data    json    not null
);