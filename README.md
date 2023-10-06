# debefix - Database seeding and fixtures
[![GoDoc](https://godoc.org/github.com/RangelReale/debefix?status.png)](https://godoc.org/github.com/RangelReale/debefix)

debefix is a Go library (and a cli in the future) to seed database data and/or create fixtures for DB tests.

Tables can reference each other using string ids (called "refid"), generated fields (like database auto increment or
generated UUID) are supported and can be resolved and used by other table's references.

Dependencies between tables can be detected automatically by reference ids, or manually. This is used to generate a
dependency graph and output the insert statements in the correct order.

Using the yaml tag `!dbfexpr` it is possible to define expressions on field values.

Tables with rows can be declared at the top-level on inside a parent row using a special `_dbfdeps` field. In this case,
values from the parent row can be used, using the `parent:<fieldname>` expression.

## Field value expressions

- `!dbfexpr "refid:<table>:<refid>:<fieldname>"`: reference a **refid** field value in a table. This id is 
  declared using a `_dbfconfig: {"refid": <refid>}` special field in the row.
- `!dbfexpr "parent:<fieldname>"`: reference a field in the parent table. This can only be used inside a `_dbfdeps` 
  block.
- `!dbfexpr "generated"`: indicates that this is a generated field that must be supplied at resolve time, and can later
  be used by other references once resolved.

## Generating SQL

SQL can be generated using `github.com/RangelReale/debefix/db/sql/<dbtype>`.

```go
import (
    "sql"

    "github.com/RangelReale/debefix"
    "github.com/RangelReale/debefix/db/sql"
    "github.com/RangelReale/debefix/db/sql/postgres"
)

func main() {
    db, err := sql.Open("postgres", "dsn://postgres")
    if err != nil {
        panic(err)
    }

    data, err := debefix.LoadDirectory("/x/y")
    if err != nil {
        panic(err)
    }

    // will send an INSERT SQL for each row to the db, taking table dependency in account for the correct order. 
    err = postgres.Resolve(sql.NewSQLQueryInterface(db), data)
    if err != nil {
        panic(err)
    }
}
```

## Generating Non-SQL

The import `github.com/RangelReale/debefix/db` contains a `ResolverFunc` that is not directly tied to SQL, it can be
used to insert data in any database that has the concepts of "tables" with a list of field/values.

As inner maps/arrays are supported by YAML, data with more complex structure should work without any problems.

## Sample input

The configuration can be in a single or multiple files, the file itself doesn't matter. The file names/directories are 
sorted alphabetically, so the order can be deterministic.

The same table can also be present in multiple files, given that the `config` section is equal (or only set in one of them).

```yaml
tags:
  config:
    table_name: "public.tag" # database table name. If not set, will use the table id (tags) as the table name.
  rows:
    - tag_id: !dbfexpr "generated" # means that this will be generated, for example as a database autoincrement
      name: "Go"
      created_at: !!timestamp 2023-01-01T12:30:12Z
      updated_at: !!timestamp 2023-01-01T12:30:12Z
      _dbfconfig:
        refid: "go" # refid to be targeted by '!dbfexpr "refid:tags:go:tag_id"'
    - tag_id: !dbfexpr "generated"
      name: "JavaScript"
      created_at: !!timestamp 2023-01-01T12:30:12Z
      updated_at: !!timestamp 2023-01-01T12:30:12Z
      _dbfconfig:
        refid: "javascript"
    - tag_id: !dbfexpr "generated"
      name: "C++"
      created_at: !!timestamp 2023-01-01T12:30:12Z
      updated_at: !!timestamp 2023-01-01T12:30:12Z
      _dbfconfig:
        refid: "cpp"
users:
  config:
    table_name: "public.user"
  rows:
    - user_id: 1
      name: "John Doe"
      email: "john@example.com"
      created_at: !!timestamp 2023-01-01T12:30:12Z
      updated_at: !!timestamp 2023-01-01T12:30:12Z
      _dbfconfig:
        refid: "johndoe" # refid to be targeted by '!dbfexpr "refid:users:johndoe:user_id"'
    - user_id: 2
      name: "Jane Doe"
      email: "jane@example.com"
      created_at: !!timestamp 2023-01-04T12:30:12Z
      updated_at: !!timestamp 2023-01-04T12:30:12Z
      _dbfconfig:
        refid: "janedoe"
posts:
  config:
    table_name: "public.post"
  rows:
    - post_id: 1
      title: "Post 1"
      text: "This is the text of the first post"
      user_id: !dbfexpr "refid:users:johndoe:user_id"
      created_at: !!timestamp 2023-01-01T12:30:12Z
      updated_at: !!timestamp 2023-01-01T12:30:12Z
      _dbfdeps:
        posts_tags: # declaring tables in _dbfdeps is exactly the same as declaring top-level, but allows using "parent" expression to get parent info
          rows:
            - post_id: !dbfexpr "parent:post_id"
              tag_id: !dbfexpr "refid:tags:go:tag_id"
      _dbfconfig:
        refid: "post_1"
    - post_id: 2
      parent_post_id: !dbfexpr "refid:posts:post_1:post_id" # order matters, so self-referential fields must be set in order
      title: "Post 2"
      text: "This is the text of the seco d post"
      user_id: !dbfexpr "refid:users:johndoe:user_id"
      created_at: !!timestamp 2023-01-02T12:30:12Z
      updated_at: !!timestamp 2023-01-02T12:30:12Z
      _dbfdeps:
        posts_tags:
          rows:
            - post_id: !dbfexpr "parent:post_id"
              tag_id: !dbfexpr "refid:tags:javascript:tag_id" # tag_id is generated so the value will be resolved before being set here 
        comments:
          rows:
            - comment_id: 3
              post_id: !dbfexpr "parent:post_id"
              user_id: !dbfexpr "refid:users:janedoe:user_id"
              text: "I liked this post!"
posts_tags:
  config:
    table_name: "public.post_tag"
comments:
  config:
    depends:
      - posts # add a manual dependency if there is no refid linking the tables
  rows:
    - comment_id: 1
      post_id: 1
      user_id: !dbfexpr "refid:users:janedoe:user_id"
      text: "Good post!"
      created_at: !!timestamp 2023-01-01T12:31:12Z
      updated_at: !!timestamp 2023-01-01T12:31:12Z
    - comment_id: 2
      post_id: 1
      user_id: !dbfexpr "refid:users:johndoe:user_id"
      text: "Thanks!"
      created_at: !!timestamp 2023-01-01T12:35:12Z
      updated_at: !!timestamp 2023-01-01T12:35:12Z
```

# License

MIT

### Author

Rangel Reale (rangelreale@gmail.com)
