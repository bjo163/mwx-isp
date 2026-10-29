# Table-level tenant-scope checker, a bounded study (FUT-010)

Status: research note and prototype, 2026-10-29 on `dev`.
Scope: `TR-F033` tenant isolation; builds on the raw-SQL ratchet from `ENG-019`.
Non-goal: replacing the production gate. The prototype is not wired into lint
or CI; the existing `TestRawSQLTenantScopeRatchet` remains the enforced rule.

## Question

`ENG-019` reduced tenant-isolation regressions to one mechanical rule: a
`Raw`/`Exec`/`Table`/`Joins` call must either contain the string `tenant_id`
somewhere in the same statement or carry a reasoned exemption. That rule is
cheap and it caught the problem it was built for, but it answers a text
question, not a scope question. Three statements pass it without bounding a
single row:

```go
db.Raw(`SELECT * FROM radius_online -- TODO tenant_id = ?`)
db.Raw(`SELECT username FROM radius_online ORDER BY tenant_id`)
db.Table("radius_online AS o").
    Joins("JOIN radius_accounting AS a ON a.username = o.username AND a.tenant_id = o.tenant_id")
```

The first two merely mention the column; the third joins two unscoped tables on
their tenant columns, which is a relational predicate, not a bound. The study
asked whether a small statement analyzer can classify the real corpus per
table, and whether it is worth enforcing.

## Method

Prototype: `internal/tenancy/policycheck` (analysis only; nothing imports it
from product code).

1. **Collect statements.** The Go AST finds every statement or package-level
   initializer containing a raw-SQL call, concatenates the string-literal
   arguments of that call chain in source order, and restores the clause
   keyword GORM strips (`Table("x")` → `FROM x`, `Where("...")` → `WHERE ...`).
   A statement whose relation or SQL text is built at runtime is reported as
   unanalyzable instead of being guessed at.
2. **Lex and classify.** A small SQL lexer drops comments and string contents,
   reads relations after `FROM`, `JOIN`, `UPDATE`, `INTO`, and scans `WHERE` and
   `ON` regions at parenthesis depth zero.
3. **Scope per table.** `alias.tenant_id = ?`, `= $1`, `= 0`, or a bare
   `tenant_id = ?` in a one-table statement is a bound. `a.tenant_id =
   b.tenant_id` is only an edge: it scopes nothing unless one side is already
   bound. Edges are followed to a fixpoint, so a three-table chain is scoped
   through its driving table and not through its joins.
4. **Remember the exemption.** The same `// tenancy:unscoped: <reason>` comment
   the production ratchet accepts exempts a site here too, so the two tools
   agree on what "deliberate" looks like.

## What the corpus looks like

`go test -v ./internal/tenancy/policycheck/ -run TestRepoRawSQLStatementsSatisfyTheTablePolicy`:

```
internal/adminapi/dashboard.go:280   unanalyzable: runtime SQL
internal/adminapi/nas.go:389         unanalyzable: model-based driving table
internal/adminapi/noc_impact.go:382  tables=[u scoped, c scoped, s scoped]  unscoped=[]
internal/adminapi/noc_impact.go:435  tables=[radius_accounting scoped]       unscoped=[]
internal/adminapi/noc_impact.go:446  tables=[radius_online scoped]           unscoped=[]
internal/adminapi/noc_impact.go:496  tables=[radius_online AS o scoped]      unscoped=[]
internal/adminapi/noc_impact.go:506  tables=[isp_trouble_ticket AS t scoped] unscoped=[]
internal/adminapi/users.go:295       unanalyzable: model-based driving table
internal/app/app.go:476              unanalyzable: no analysable relation
internal/app/app.go:578              unanalyzable: runtime SQL
internal/app/app.go:605              unanalyzable: runtime SQL
literal=5 dynamic=6 annotated=0
```

- 11 raw-SQL statements; 5 carry analyzable text and all 5 bound every
  relation, including the three-table `noc_impact` chain that is scoped through
  `s.tenant_id` and reached transitively through the join equalities.
- 6 cannot be analyzed: 3 assemble SQL at runtime, 2 take their driving table
  from `Model(&domain.X{})` and are scoped by the tenant-aware GORM callback,
  and 1 (`setval` on the tenant registry) contains its only `FROM` inside a
  nested call and is already exempted with a reason.
- The gate therefore passes today. The value of the prototype is that it would
  fail on each of the three examples at the top of this note, which the string
  ratchet accepts.

## Decision

1. **Keep the string ratchet as the enforced gate.** It needs no SQL parsing,
   runs in the existing unit lane, and covers runtime-assembled statements
   equally well because it only demands the column name.
2. **Keep the table-level checker as a prototype.** Five of eleven statements
   are visible to it and six are not, so wiring it in today would add a second,
   weaker gate over less than half the corpus.
3. **Revisit when either condition changes:** the number of literal raw
   statements grows past a handful, or the "unanalyzable" count drops because
   `Model`/runtime statements are rewritten as explicit `Table` chains. At that
   point `Findings()` is already the predicate a lint step needs.

## Limits of the prototype

- Parenthesized subqueries and CTEs are not descended into; their relations are
  the owning statement's responsibility. Derived-table joins such as
  `LEFT JOIN (SELECT … ) ro ON …` therefore contribute no relation.
- `Model(&domain.X{})` is not mapped to a table name, so model-based statements
  are opaque. That is a deliberate omission: their scope comes from the GORM
  callback chain, not from the SQL text.
- A non-literal `Where`/`Having` fragment (for example `Where(sel.where)`) is
  skipped rather than flagged; only relation-introducing arguments make a
  statement opaque.
- `INSERT … SELECT`, `UPDATE … FROM`, `USING`, and set operations are treated as
  plain relation lists; the analyzer bounds tables, it does not model semantics.
- A bare `tenant_id` is accepted only in a one-table statement; in a join it is
  reported as ambiguous, which is the honest reading but also a false positive
  for a query whose unqualified column is unique.
- Table names are compared literally; an alias that shadows a table name is not
  distinguished from it.

## Verification

```
go test -count=1 ./internal/tenancy/policycheck/
ok  github.com/bjo163/mwx-isp/internal/tenancy/policycheck  0.860s

golangci-lint run ./internal/tenancy/policycheck/...
0 issues.
```

Fixture coverage (`TestAnalyzeScopeFixtures`): a bound predicate, no predicate,
`tenant_id` only in a comment, `tenant_id` only in `ORDER BY`, a join chain
scoped from the driving table, two unscoped tables joined on their tenant
columns, a three-table transitive chain, an edge from an unscoped table, an
ambiguous bare `tenant_id`, `DELETE`/`UPDATE` with and without a bound,
quoted/schema-qualified identifiers, a subquery, and a statement with no tables.

`TestScanFileSeparatesLiteralFromDynamic` pins the Go-side extraction: literal,
annotated, runtime, and chained-`Where` statements are classified separately,
and only the unannotated unscoped literal becomes a finding.
