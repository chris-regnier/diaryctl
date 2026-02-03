# Context Providers Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add first-class context objects and content providers to diaryctl so entries can be semantically grouped and editor buffers can be pre-populated with contextual information.

**Architecture:** New `internal/context/` package defines `ContentProvider` and `ContextResolver` interfaces. `Context` is a storage-level type (like `Template`). `ContextRef` is added to `entry.Entry` (like `TemplateRef`). Both storage backends get context CRUD + join table. CLI gets `context` subcommand group. Existing `create`/`jot`/`today` commands gain automatic context attachment.

**Tech Stack:** Go 1.24, Cobra (CLI), Viper (config), standard library `os/exec` for git, `encoding/json` for state file.

---

### Task 1: Add ContextRef to entry package

**Files:**
- Modify: `internal/entry/entry.go`

**Step 1: Write the failing test**

Create `internal/entry/entry_test.go` (if it doesn't exist) or add to existing:

```go
func TestEntryContextRefJSON(t *testing.T) {
	e := Entry{
		ID:      "abc12345",
		Content: "test",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
		Contexts: []ContextRef{
			{ContextID: "ctx00001", ContextName: "feature/auth"},
		},
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Entry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Contexts) != 1 || got.Contexts[0].ContextName != "feature/auth" {
		t.Errorf("got contexts %v", got.Contexts)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/entry/ -run TestEntryContextRefJSON -v`
Expected: FAIL — `ContextRef` undefined

**Step 3: Write minimal implementation**

Add to `internal/entry/entry.go`:

```go
// ContextRef is a lightweight reference to a context, stored on entries for grouping.
type ContextRef struct {
	ContextID   string `json:"context_id"`
	ContextName string `json:"context_name"`
}
```

Add field to `Entry` struct:

```go
Contexts  []ContextRef  `json:"contexts,omitempty"`
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/entry/ -run TestEntryContextRefJSON -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/entry/
git commit -m "feat: add ContextRef type and Contexts field to Entry"
```

---

### Task 2: Add Context type and context methods to Storage interface

**Files:**
- Modify: `internal/storage/storage.go`

**Step 1: Add Context type and interface methods**

Add to `internal/storage/storage.go` after the `Template` struct:

```go
// Context represents a semantic grouping for diary entries.
type Context struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Source    string    `json:"source"` // "manual", "git", etc.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

Add to the `Storage` interface after template methods:

```go
	// Context methods
	CreateContext(c Context) error
	GetContext(id string) (Context, error)
	GetContextByName(name string) (Context, error)
	ListContexts() ([]Context, error)
	DeleteContext(id string) error
	AttachContext(entryID string, contextID string) error
	DetachContext(entryID string, contextID string) error
```

Add `ContextName` field to `ListOptions`:

```go
	ContextName  string     // filter entries by context name
```

**Step 2: Verify compilation fails (backends don't implement new methods yet)**

Run: `go build ./...`
Expected: FAIL — SQLite and Markdown stores don't implement the new interface methods.

**Step 3: Commit (interface-only, backends will follow)**

```bash
git add internal/storage/storage.go
git commit -m "feat: add Context type and context methods to Storage interface"
```

---

### Task 3: Implement context methods in SQLite backend

**Files:**
- Modify: `internal/storage/sqlite/sqlite.go`

**Step 1: Add schema for contexts**

In `createSchema`, add these statements to the `statements` slice:

```go
`CREATE TABLE IF NOT EXISTS contexts (
	id         TEXT PRIMARY KEY,
	name       TEXT NOT NULL UNIQUE,
	source     TEXT NOT NULL DEFAULT 'manual',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
)`,
`CREATE TABLE IF NOT EXISTS entry_contexts (
	entry_id    TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
	context_id  TEXT NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
	PRIMARY KEY (entry_id, context_id)
)`,
```

**Step 2: Implement CreateContext**

```go
func (s *Store) CreateContext(c storage.Context) error {
	_, err := s.db.Exec(
		"INSERT INTO contexts (id, name, source, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		c.ID, c.Name, c.Source,
		c.CreatedAt.UTC().Format(time.RFC3339),
		c.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("%w: context name %q already exists", storage.ErrConflict, c.Name)
		}
		return fmt.Errorf("%w: inserting context: %v", storage.ErrStorage, err)
	}
	return nil
}
```

**Step 3: Implement GetContext**

```go
func (s *Store) GetContext(id string) (storage.Context, error) {
	row := s.db.QueryRow("SELECT id, name, source, created_at, updated_at FROM contexts WHERE id = ?", id)
	return s.scanContext(row)
}
```

**Step 4: Implement GetContextByName**

```go
func (s *Store) GetContextByName(name string) (storage.Context, error) {
	row := s.db.QueryRow("SELECT id, name, source, created_at, updated_at FROM contexts WHERE name = ?", name)
	return s.scanContext(row)
}
```

**Step 5: Implement scanContext helper**

```go
func (s *Store) scanContext(row *sql.Row) (storage.Context, error) {
	var c storage.Context
	var createdStr, updatedStr string
	if err := row.Scan(&c.ID, &c.Name, &c.Source, &createdStr, &updatedStr); err != nil {
		if err == sql.ErrNoRows {
			return storage.Context{}, storage.ErrNotFound
		}
		return storage.Context{}, fmt.Errorf("%w: scanning context: %v", storage.ErrStorage, err)
	}
	var err error
	c.CreatedAt, err = time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return storage.Context{}, fmt.Errorf("%w: parsing created_at: %v", storage.ErrStorage, err)
	}
	c.UpdatedAt, err = time.Parse(time.RFC3339, updatedStr)
	if err != nil {
		return storage.Context{}, fmt.Errorf("%w: parsing updated_at: %v", storage.ErrStorage, err)
	}
	return c, nil
}
```

**Step 6: Implement ListContexts**

```go
func (s *Store) ListContexts() ([]storage.Context, error) {
	rows, err := s.db.Query("SELECT id, name, source, created_at, updated_at FROM contexts ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("%w: listing contexts: %v", storage.ErrStorage, err)
	}
	defer rows.Close()

	var contexts []storage.Context
	for rows.Next() {
		var c storage.Context
		var createdStr, updatedStr string
		if err := rows.Scan(&c.ID, &c.Name, &c.Source, &createdStr, &updatedStr); err != nil {
			return nil, fmt.Errorf("%w: scanning context: %v", storage.ErrStorage, err)
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		contexts = append(contexts, c)
	}
	return contexts, rows.Err()
}
```

**Step 7: Implement DeleteContext**

```go
func (s *Store) DeleteContext(id string) error {
	result, err := s.db.Exec("DELETE FROM contexts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("%w: deleting context: %v", storage.ErrStorage, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}
```

**Step 8: Implement AttachContext and DetachContext**

```go
func (s *Store) AttachContext(entryID string, contextID string) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO entry_contexts (entry_id, context_id) VALUES (?, ?)",
		entryID, contextID,
	)
	if err != nil {
		return fmt.Errorf("%w: attaching context: %v", storage.ErrStorage, err)
	}
	return nil
}

func (s *Store) DetachContext(entryID string, contextID string) error {
	_, err := s.db.Exec(
		"DELETE FROM entry_contexts WHERE entry_id = ? AND context_id = ?",
		entryID, contextID,
	)
	if err != nil {
		return fmt.Errorf("%w: detaching context: %v", storage.ErrStorage, err)
	}
	return nil
}
```

**Step 9: Implement loadContextRefs** (mirrors `loadTemplateRefs`)

```go
func (s *Store) loadContextRefs(entryID string) ([]entry.ContextRef, error) {
	rows, err := s.db.Query(
		"SELECT c.id, c.name FROM entry_contexts ec JOIN contexts c ON c.id = ec.context_id WHERE ec.entry_id = ?",
		entryID,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: querying context refs: %v", storage.ErrStorage, err)
	}
	defer rows.Close()

	var refs []entry.ContextRef
	for rows.Next() {
		var ref entry.ContextRef
		if err := rows.Scan(&ref.ContextID, &ref.ContextName); err != nil {
			return nil, fmt.Errorf("%w: scanning context ref: %v", storage.ErrStorage, err)
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}
```

**Step 10: Wire loadContextRefs into Get**

In the `Get` method, after loading template refs, add:

```go
	// Load context refs
	ctxRefs, err := s.loadContextRefs(id)
	if err != nil {
		return entry.Entry{}, err
	}
	e.Contexts = ctxRefs
```

**Step 11: Wire loadContextRefs into List**

In the `scanEntries` helper (or wherever entries are scanned in List), load context refs for each entry. Also add the `ContextName` filter join:

```go
if opts.ContextName != "" {
	query += " JOIN entry_contexts ec ON ec.entry_id = entries.id JOIN contexts ctx ON ctx.id = ec.context_id"
	conditions = append(conditions, "ctx.name = ?")
	args = append(args, opts.ContextName)
}
```

**Step 12: Wire context refs into Create**

In the `Create` method, after inserting template refs, add:

```go
	for _, ref := range e.Contexts {
		_, err = tx.Exec(
			"INSERT OR IGNORE INTO entry_contexts (entry_id, context_id) VALUES (?, ?)",
			e.ID, ref.ContextID,
		)
		if err != nil {
			return fmt.Errorf("%w: inserting context ref: %v", storage.ErrStorage, err)
		}
	}
```

**Step 13: Verify compilation**

Run: `go build ./internal/storage/sqlite/`
Expected: PASS

**Step 14: Commit**

```bash
git add internal/storage/sqlite/
git commit -m "feat: implement context methods in SQLite backend"
```

---

### Task 4: Implement context methods in Markdown backend

**Files:**
- Modify: `internal/storage/markdown/markdown.go`

**Step 1: Add contexts dir to Store struct and New()**

In `New()`, add:

```go
	contextsDir := filepath.Join(dataDir, "contexts")
	if err := os.MkdirAll(contextsDir, 0755); err != nil {
		return nil, fmt.Errorf("%w: creating contexts directory: %v", storage.ErrStorage, err)
	}
```

Add `contextsDir string` to the `Store` struct. Return it in the constructor.

**Step 2: Add context fields to frontmatter**

Add to `frontMatter` struct:

```go
	Contexts  []fmContextRef  `yaml:"contexts"`
```

Add new struct:

```go
type fmContextRef struct {
	ContextID   string `yaml:"context_id"`
	ContextName string `yaml:"context_name"`
}
```

**Step 3: Update marshal to write context refs**

After template marshalling in `marshal()`:

```go
	if len(e.Contexts) > 0 {
		b.WriteString("contexts:\n")
		for _, ref := range e.Contexts {
			fmt.Fprintf(&b, "  - context_id: %s\n", ref.ContextID)
			fmt.Fprintf(&b, "    context_name: %s\n", ref.ContextName)
		}
	}
```

**Step 4: Update unmarshal to read context refs**

After template ref unmarshalling in `unmarshal()`:

```go
	var contexts []entry.ContextRef
	for _, ref := range fm.Contexts {
		contexts = append(contexts, entry.ContextRef{
			ContextID:   ref.ContextID,
			ContextName: ref.ContextName,
		})
	}
```

Set `Contexts: contexts` on the returned Entry.

**Step 5: Implement context CRUD using JSON files**

Context objects stored as `<contextsDir>/<id>.json`:

```go
func (s *Store) CreateContext(c storage.Context) error {
	// Check for duplicate name
	existing, _ := s.GetContextByName(c.Name)
	if existing.ID != "" {
		return fmt.Errorf("%w: context name %q already exists", storage.ErrConflict, c.Name)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: marshalling context: %v", storage.ErrStorage, err)
	}
	path := filepath.Join(s.contextsDir, c.ID+".json")
	return s.atomicWrite(path, data)
}

func (s *Store) GetContext(id string) (storage.Context, error) {
	path := filepath.Join(s.contextsDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return storage.Context{}, storage.ErrNotFound
		}
		return storage.Context{}, fmt.Errorf("%w: reading context: %v", storage.ErrStorage, err)
	}
	var c storage.Context
	if err := json.Unmarshal(data, &c); err != nil {
		return storage.Context{}, fmt.Errorf("%w: unmarshalling context: %v", storage.ErrStorage, err)
	}
	return c, nil
}

func (s *Store) GetContextByName(name string) (storage.Context, error) {
	contexts, err := s.ListContexts()
	if err != nil {
		return storage.Context{}, err
	}
	for _, c := range contexts {
		if c.Name == name {
			return c, nil
		}
	}
	return storage.Context{}, storage.ErrNotFound
}

func (s *Store) ListContexts() ([]storage.Context, error) {
	entries, err := os.ReadDir(s.contextsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: reading contexts dir: %v", storage.ErrStorage, err)
	}
	var contexts []storage.Context
	for _, de := range entries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(de.Name(), ".json")
		c, err := s.GetContext(id)
		if err != nil {
			continue
		}
		contexts = append(contexts, c)
	}
	sort.Slice(contexts, func(i, j int) bool { return contexts[i].Name < contexts[j].Name })
	return contexts, nil
}

func (s *Store) DeleteContext(id string) error {
	path := filepath.Join(s.contextsDir, id+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return storage.ErrNotFound
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("%w: deleting context: %v", storage.ErrStorage, err)
	}
	return nil
}
```

**Step 6: Implement AttachContext and DetachContext**

For markdown, these update the entry's frontmatter:

```go
func (s *Store) AttachContext(entryID string, contextID string) error {
	e, err := s.Get(entryID)
	if err != nil {
		return err
	}
	// Check if already attached
	for _, ref := range e.Contexts {
		if ref.ContextID == contextID {
			return nil // idempotent
		}
	}
	c, err := s.GetContext(contextID)
	if err != nil {
		return err
	}
	e.Contexts = append(e.Contexts, entry.ContextRef{
		ContextID: contextID, ContextName: c.Name,
	})
	return s.atomicWrite(s.entryPath(e), s.marshal(e))
}

func (s *Store) DetachContext(entryID string, contextID string) error {
	e, err := s.Get(entryID)
	if err != nil {
		return err
	}
	var filtered []entry.ContextRef
	for _, ref := range e.Contexts {
		if ref.ContextID != contextID {
			filtered = append(filtered, ref)
		}
	}
	e.Contexts = filtered
	return s.atomicWrite(s.entryPath(e), s.marshal(e))
}
```

**Step 7: Add ContextName filter to List**

In the `List` method, after template filtering logic, add context name filtering. For markdown this means checking `e.Contexts` during the entry scan loop:

```go
if opts.ContextName != "" {
	matched := false
	for _, ref := range e.Contexts {
		if ref.ContextName == opts.ContextName {
			matched = true
			break
		}
	}
	if !matched {
		continue
	}
}
```

**Step 8: Verify compilation**

Run: `go build ./internal/storage/markdown/`
Expected: PASS

**Step 9: Commit**

```bash
git add internal/storage/markdown/
git commit -m "feat: implement context methods in Markdown backend"
```

---

### Task 5: Add context contract tests

**Files:**
- Modify: `internal/storage/contract_test.go`

**Step 1: Add makeContext helper**

```go
func makeContext(t *testing.T, name, source string) storage.Context {
	t.Helper()
	id, err := entry.NewID()
	if err != nil {
		t.Fatalf("generating ID: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	return storage.Context{
		ID:        id,
		Name:      name,
		Source:    source,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
```

**Step 2: Write runContextContractTests**

```go
func runContextContractTests(t *testing.T, name string, factory storageFactory) {
	t.Run(name+" Context CRUD", func(t *testing.T) {

		t.Run("CreateContext and GetContext", func(t *testing.T) {
			s := factory(t)
			ctx := makeContext(t, "feature/auth", "git")
			if err := s.CreateContext(ctx); err != nil {
				t.Fatalf("CreateContext: %v", err)
			}
			got, err := s.GetContext(ctx.ID)
			if err != nil {
				t.Fatalf("GetContext: %v", err)
			}
			if got.Name != "feature/auth" || got.Source != "git" {
				t.Errorf("got name=%q source=%q", got.Name, got.Source)
			}
		})

		t.Run("GetContextByName", func(t *testing.T) {
			s := factory(t)
			ctx := makeContext(t, "sprint:23", "manual")
			_ = s.CreateContext(ctx)
			got, err := s.GetContextByName("sprint:23")
			if err != nil {
				t.Fatalf("GetContextByName: %v", err)
			}
			if got.ID != ctx.ID {
				t.Errorf("got ID=%q want %q", got.ID, ctx.ID)
			}
		})

		t.Run("GetContextByName not found", func(t *testing.T) {
			s := factory(t)
			_, err := s.GetContextByName("nonexistent")
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})

		t.Run("CreateContext duplicate name", func(t *testing.T) {
			s := factory(t)
			ctx1 := makeContext(t, "dup-name", "manual")
			ctx2 := makeContext(t, "dup-name", "git")
			_ = s.CreateContext(ctx1)
			err := s.CreateContext(ctx2)
			if !errors.Is(err, storage.ErrConflict) {
				t.Errorf("expected ErrConflict, got %v", err)
			}
		})

		t.Run("ListContexts", func(t *testing.T) {
			s := factory(t)
			_ = s.CreateContext(makeContext(t, "alpha", "manual"))
			_ = s.CreateContext(makeContext(t, "beta", "git"))
			_ = s.CreateContext(makeContext(t, "gamma", "manual"))
			list, err := s.ListContexts()
			if err != nil {
				t.Fatalf("ListContexts: %v", err)
			}
			if len(list) != 3 {
				t.Errorf("got %d contexts, want 3", len(list))
			}
		})

		t.Run("DeleteContext", func(t *testing.T) {
			s := factory(t)
			ctx := makeContext(t, "to-delete", "manual")
			_ = s.CreateContext(ctx)
			if err := s.DeleteContext(ctx.ID); err != nil {
				t.Fatalf("DeleteContext: %v", err)
			}
			_, err := s.GetContext(ctx.ID)
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound after delete, got %v", err)
			}
		})

		t.Run("DeleteContext not found", func(t *testing.T) {
			s := factory(t)
			err := s.DeleteContext("nonexist")
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})

		t.Run("AttachContext and load on Get", func(t *testing.T) {
			s := factory(t)
			ctx := makeContext(t, "feature/auth", "git")
			_ = s.CreateContext(ctx)
			e := makeEntry(t, "hello world")
			_ = s.Create(e)
			if err := s.AttachContext(e.ID, ctx.ID); err != nil {
				t.Fatalf("AttachContext: %v", err)
			}
			got, err := s.Get(e.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if len(got.Contexts) != 1 || got.Contexts[0].ContextName != "feature/auth" {
				t.Errorf("expected 1 context ref 'feature/auth', got %v", got.Contexts)
			}
		})

		t.Run("AttachContext idempotent", func(t *testing.T) {
			s := factory(t)
			ctx := makeContext(t, "feature/auth", "git")
			_ = s.CreateContext(ctx)
			e := makeEntry(t, "hello")
			_ = s.Create(e)
			_ = s.AttachContext(e.ID, ctx.ID)
			_ = s.AttachContext(e.ID, ctx.ID) // second attach
			got, _ := s.Get(e.ID)
			if len(got.Contexts) != 1 {
				t.Errorf("expected 1 context after double attach, got %d", len(got.Contexts))
			}
		})

		t.Run("DetachContext", func(t *testing.T) {
			s := factory(t)
			ctx := makeContext(t, "feature/auth", "git")
			_ = s.CreateContext(ctx)
			e := makeEntry(t, "hello")
			_ = s.Create(e)
			_ = s.AttachContext(e.ID, ctx.ID)
			if err := s.DetachContext(e.ID, ctx.ID); err != nil {
				t.Fatalf("DetachContext: %v", err)
			}
			got, _ := s.Get(e.ID)
			if len(got.Contexts) != 0 {
				t.Errorf("expected 0 contexts after detach, got %d", len(got.Contexts))
			}
		})

		t.Run("Create entry with context refs", func(t *testing.T) {
			s := factory(t)
			ctx := makeContext(t, "feature/auth", "git")
			_ = s.CreateContext(ctx)
			e := makeEntry(t, "with context")
			e.Contexts = []entry.ContextRef{
				{ContextID: ctx.ID, ContextName: ctx.Name},
			}
			_ = s.Create(e)
			got, _ := s.Get(e.ID)
			if len(got.Contexts) != 1 || got.Contexts[0].ContextName != "feature/auth" {
				t.Errorf("expected 1 context 'feature/auth', got %v", got.Contexts)
			}
		})

		t.Run("List filtered by context name", func(t *testing.T) {
			s := factory(t)
			ctx := makeContext(t, "feature/auth", "git")
			_ = s.CreateContext(ctx)

			e1 := makeEntry(t, "with context")
			_ = s.Create(e1)
			_ = s.AttachContext(e1.ID, ctx.ID)

			e2 := makeEntry(t, "without context")
			_ = s.Create(e2)

			results, err := s.List(storage.ListOptions{ContextName: "feature/auth"})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(results) != 1 || results[0].ID != e1.ID {
				t.Errorf("expected 1 entry with context 'feature/auth', got %d", len(results))
			}
		})

		t.Run("DeleteContext detaches from entries", func(t *testing.T) {
			s := factory(t)
			ctx := makeContext(t, "feature/auth", "git")
			_ = s.CreateContext(ctx)
			e := makeEntry(t, "hello")
			_ = s.Create(e)
			_ = s.AttachContext(e.ID, ctx.ID)
			_ = s.DeleteContext(ctx.ID)
			got, _ := s.Get(e.ID)
			if len(got.Contexts) != 0 {
				t.Errorf("expected 0 contexts after context deletion, got %d", len(got.Contexts))
			}
		})
	})
}
```

**Step 3: Register the new contract test suite**

Add to `TestMarkdownStorage`:
```go
runContextContractTests(t, "Markdown", markdownFactory)
```

Add to `TestSQLiteStorage`:
```go
runContextContractTests(t, "SQLite", sqliteFactory)
```

**Step 4: Run tests**

Run: `go test ./internal/storage/ -v -run "Context"`
Expected: PASS for both backends

**Step 5: Commit**

```bash
git add internal/storage/contract_test.go
git commit -m "test: add context contract tests for both storage backends"
```

---

### Task 6: Add config fields for context providers and resolvers

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add fields to Config struct**

```go
	ContextProviders []string `mapstructure:"context_providers"`
	ContextResolvers []string `mapstructure:"context_resolvers"`
```

**Step 2: Add defaults in Load()**

```go
	v.SetDefault("context_providers", []string{})
	v.SetDefault("context_resolvers", []string{})
```

**Step 3: Verify compilation**

Run: `go build ./internal/config/`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/config/
git commit -m "feat: add context_providers and context_resolvers config fields"
```

---

### Task 7: Create internal/context package with interfaces and composition

**Files:**
- Create: `internal/context/context.go`
- Create: `internal/context/context_test.go`

**Step 1: Write failing test for ComposeContent**

Create `internal/context/context_test.go`:

```go
package context

import "testing"

type stubProvider struct {
	name   string
	output string
	err    error
}

func (s *stubProvider) Name() string              { return s.name }
func (s *stubProvider) Generate() (string, error)  { return s.output, s.err }

func TestComposeContent_empty(t *testing.T) {
	got := ComposeContent(nil, "template content")
	if got != "template content" {
		t.Errorf("got %q, want %q", got, "template content")
	}
}

func TestComposeContent_providersOnly(t *testing.T) {
	providers := []ContentProvider{
		&stubProvider{name: "datetime", output: "# Monday, February 2, 2026"},
	}
	got := ComposeContent(providers, "")
	if got != "# Monday, February 2, 2026" {
		t.Errorf("got %q", got)
	}
}

func TestComposeContent_providersAndTemplate(t *testing.T) {
	providers := []ContentProvider{
		&stubProvider{name: "datetime", output: "# Monday, February 2, 2026"},
		&stubProvider{name: "git", output: "branch: main | 0 uncommitted files"},
	}
	got := ComposeContent(providers, "## Notes")
	want := "# Monday, February 2, 2026\nbranch: main | 0 uncommitted files\n\n## Notes"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestComposeContent_skipsEmptyProviders(t *testing.T) {
	providers := []ContentProvider{
		&stubProvider{name: "datetime", output: "# Monday"},
		&stubProvider{name: "git", output: ""},
	}
	got := ComposeContent(providers, "## Notes")
	want := "# Monday\n\n## Notes"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestComposeContent_skipsFailedProviders(t *testing.T) {
	providers := []ContentProvider{
		&stubProvider{name: "datetime", output: "# Monday"},
		&stubProvider{name: "git", output: "", err: fmt.Errorf("not a git repo")},
	}
	got := ComposeContent(providers, "## Notes")
	want := "# Monday\n\n## Notes"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/context/ -v`
Expected: FAIL — package doesn't exist

**Step 3: Implement interfaces and ComposeContent**

Create `internal/context/context.go`:

```go
package context

import "strings"

// ContentProvider generates text for the editor buffer.
type ContentProvider interface {
	Name() string
	Generate() (string, error)
}

// ContextResolver detects contexts from the environment.
type ContextResolver interface {
	Name() string
	Resolve() ([]string, error)
}

// ComposeContent assembles the editor buffer from provider output and template content.
// Order: provider outputs (in order, separated by newlines) → blank line → template content.
// Providers that return empty strings or errors are silently skipped.
func ComposeContent(providers []ContentProvider, templateContent string) string {
	var parts []string
	for _, p := range providers {
		output, err := p.Generate()
		if err != nil || output == "" {
			continue
		}
		parts = append(parts, output)
	}

	providerText := strings.Join(parts, "\n")

	switch {
	case providerText == "" && templateContent == "":
		return ""
	case providerText == "":
		return templateContent
	case templateContent == "":
		return providerText
	default:
		return providerText + "\n\n" + templateContent
	}
}
```

**Step 4: Run tests**

Run: `go test ./internal/context/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/context/
git commit -m "feat: add context package with ContentProvider, ContextResolver interfaces and ComposeContent"
```

---

### Task 8: Implement ResolveActiveContexts

**Files:**
- Modify: `internal/context/context.go`
- Modify: `internal/context/context_test.go`

**Step 1: Write failing test**

Add to `internal/context/context_test.go`:

```go
type stubResolver struct {
	name    string
	names   []string
	err     error
}

func (s *stubResolver) Name() string              { return s.name }
func (s *stubResolver) Resolve() ([]string, error) { return s.names, s.err }

// mockContextStore is a minimal mock for the ContextStore interface.
type mockContextStore struct {
	contexts map[string]storage.Context // keyed by name
}

func (m *mockContextStore) GetContextByName(name string) (storage.Context, error) {
	c, ok := m.contexts[name]
	if !ok {
		return storage.Context{}, storage.ErrNotFound
	}
	return c, nil
}

func (m *mockContextStore) CreateContext(c storage.Context) error {
	m.contexts[c.Name] = c
	return nil
}

func TestResolveActiveContexts_empty(t *testing.T) {
	ms := &mockContextStore{contexts: map[string]storage.Context{}}
	refs, err := ResolveActiveContexts(nil, nil, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

func TestResolveActiveContexts_manualOnly(t *testing.T) {
	ms := &mockContextStore{contexts: map[string]storage.Context{}}
	refs, err := ResolveActiveContexts(nil, []string{"sprint:23"}, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 1 || refs[0].ContextName != "sprint:23" {
		t.Errorf("got refs %v", refs)
	}
	// Should have auto-created the context
	if _, ok := ms.contexts["sprint:23"]; !ok {
		t.Error("expected context to be auto-created")
	}
}

func TestResolveActiveContexts_resolverOnly(t *testing.T) {
	ms := &mockContextStore{contexts: map[string]storage.Context{}}
	resolvers := []ContextResolver{
		&stubResolver{name: "git", names: []string{"feature/auth"}},
	}
	refs, err := ResolveActiveContexts(resolvers, nil, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 1 || refs[0].ContextName != "feature/auth" {
		t.Errorf("got refs %v", refs)
	}
}

func TestResolveActiveContexts_deduplicates(t *testing.T) {
	ms := &mockContextStore{contexts: map[string]storage.Context{}}
	resolvers := []ContextResolver{
		&stubResolver{name: "git", names: []string{"feature/auth"}},
	}
	refs, err := ResolveActiveContexts(resolvers, []string{"feature/auth"}, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 1 {
		t.Errorf("expected 1 deduplicated ref, got %d", len(refs))
	}
}

func TestResolveActiveContexts_skipsFailedResolver(t *testing.T) {
	ms := &mockContextStore{contexts: map[string]storage.Context{}}
	resolvers := []ContextResolver{
		&stubResolver{name: "git", names: nil, err: fmt.Errorf("not a git repo")},
	}
	refs, err := ResolveActiveContexts(resolvers, []string{"sprint:23"}, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 1 || refs[0].ContextName != "sprint:23" {
		t.Errorf("got refs %v", refs)
	}
}

func TestResolveActiveContexts_reusesExisting(t *testing.T) {
	existing := storage.Context{ID: "existing1", Name: "sprint:23", Source: "manual"}
	ms := &mockContextStore{contexts: map[string]storage.Context{"sprint:23": existing}}
	refs, err := ResolveActiveContexts(nil, []string{"sprint:23"}, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 1 || refs[0].ContextID != "existing1" {
		t.Errorf("expected existing context ID, got %v", refs)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/context/ -run TestResolveActiveContexts -v`
Expected: FAIL — `ResolveActiveContexts` undefined

**Step 3: Implement ContextStore interface and ResolveActiveContexts**

Add to `internal/context/context.go`:

```go
import (
	"strings"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// ContextStore is the subset of storage.Storage needed for context resolution.
type ContextStore interface {
	GetContextByName(name string) (storage.Context, error)
	CreateContext(c storage.Context) error
}

// ResolveActiveContexts gathers contexts from resolvers and manual list,
// deduplicates, and ensures each exists in storage (creating if needed).
// Failed resolvers are silently skipped.
func ResolveActiveContexts(
	resolvers []ContextResolver,
	manualContexts []string,
	store ContextStore,
) ([]entry.ContextRef, error) {
	// Collect all context names
	seen := make(map[string]string) // name -> source

	for _, r := range resolvers {
		names, err := r.Resolve()
		if err != nil {
			continue // skip failed resolvers
		}
		for _, name := range names {
			if name != "" {
				seen[name] = r.Name()
			}
		}
	}

	for _, name := range manualContexts {
		if name != "" {
			if _, exists := seen[name]; !exists {
				seen[name] = "manual"
			}
		}
	}

	if len(seen) == 0 {
		return nil, nil
	}

	// Resolve each name to a context object
	var refs []entry.ContextRef
	for name, source := range seen {
		c, err := store.GetContextByName(name)
		if err != nil {
			// Auto-create
			id, idErr := entry.NewID()
			if idErr != nil {
				continue
			}
			now := time.Now().UTC()
			c = storage.Context{
				ID:        id,
				Name:      name,
				Source:    source,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if createErr := store.CreateContext(c); createErr != nil {
				continue
			}
		}
		refs = append(refs, entry.ContextRef{
			ContextID:   c.ID,
			ContextName: c.Name,
		})
	}

	// Sort for deterministic output
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].ContextName < refs[j].ContextName
	})

	return refs, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/context/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/context/
git commit -m "feat: implement ResolveActiveContexts with auto-creation and deduplication"
```

---

### Task 9: Implement datetime content provider

**Files:**
- Create: `internal/context/datetime/datetime.go`
- Create: `internal/context/datetime/datetime_test.go`

**Step 1: Write failing test**

Create `internal/context/datetime/datetime_test.go`:

```go
package datetime

import (
	"strings"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	p := &Provider{now: func() time.Time {
		return time.Date(2026, 2, 2, 10, 30, 0, 0, time.Local)
	}}
	got, err := p.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "# Monday, February 2, 2026"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestName(t *testing.T) {
	p := New()
	if p.Name() != "datetime" {
		t.Errorf("got name %q", p.Name())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/context/datetime/ -v`
Expected: FAIL

**Step 3: Implement**

Create `internal/context/datetime/datetime.go`:

```go
package datetime

import "time"

// Provider generates a formatted date header.
type Provider struct {
	now func() time.Time // injectable for testing
}

// New creates a new datetime content provider.
func New() *Provider {
	return &Provider{now: time.Now}
}

func (p *Provider) Name() string { return "datetime" }

func (p *Provider) Generate() (string, error) {
	t := p.now()
	return "# " + t.Format("Monday, January 2, 2006"), nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/context/datetime/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/context/datetime/
git commit -m "feat: implement datetime content provider"
```

---

### Task 10: Implement git content provider and context resolver

**Files:**
- Create: `internal/context/git/git.go`
- Create: `internal/context/git/git_test.go`

**Step 1: Write tests**

Create `internal/context/git/git_test.go`:

```go
package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-b", "main")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644)
	run("add", ".")
	run("commit", "-m", "initial commit")
	return dir
}

func TestContentProvider_InRepo(t *testing.T) {
	dir := setupGitRepo(t)
	p := &ContentProvider{dir: dir}
	out, err := p.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "branch: main") {
		t.Errorf("expected branch info, got %q", out)
	}
	if !strings.Contains(out, "initial commit") {
		t.Errorf("expected commit info, got %q", out)
	}
}

func TestContentProvider_NotARepo(t *testing.T) {
	p := &ContentProvider{dir: t.TempDir()}
	out, err := p.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty string for non-repo, got %q", out)
	}
}

func TestContextResolver_InRepo(t *testing.T) {
	dir := setupGitRepo(t)
	r := &ContextResolver{dir: dir}
	names, err := r.Resolve()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "main" {
		t.Errorf("expected [main], got %v", names)
	}
}

func TestContextResolver_OnBranch(t *testing.T) {
	dir := setupGitRepo(t)
	cmd := exec.Command("git", "checkout", "-b", "feature/auth")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git checkout: %v\n%s", err, out)
	}
	r := &ContextResolver{dir: dir}
	names, err := r.Resolve()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "feature/auth" {
		t.Errorf("expected [feature/auth], got %v", names)
	}
}

func TestContextResolver_NotARepo(t *testing.T) {
	r := &ContextResolver{dir: t.TempDir()}
	names, err := r.Resolve()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty for non-repo, got %v", names)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/context/git/ -v`
Expected: FAIL

**Step 3: Implement**

Create `internal/context/git/git.go`:

```go
package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ContentProvider generates git status text for the editor buffer.
type ContentProvider struct {
	dir string // working directory; empty = current dir
}

// NewContentProvider creates a git content provider.
func NewContentProvider() *ContentProvider {
	return &ContentProvider{}
}

func (p *ContentProvider) Name() string { return "git" }

func (p *ContentProvider) Generate() (string, error) {
	branch := p.runGit("rev-parse", "--abbrev-ref", "HEAD")
	if branch == "" {
		return "", nil // not a git repo
	}

	// Count uncommitted files
	status := p.runGit("status", "--porcelain")
	dirtyCount := 0
	if status != "" {
		dirtyCount = len(strings.Split(strings.TrimSpace(status), "\n"))
	}

	line1 := fmt.Sprintf("branch: %s | %d uncommitted files", branch, dirtyCount)

	// Most recent commit
	log := p.runGit("log", "-1", "--format=%h %s (%ar)")
	if log == "" {
		return line1, nil
	}

	return line1 + "\n" + "latest: " + log, nil
}

// ContextResolver detects the current git branch as a context.
type ContextResolver struct {
	dir string
}

// NewContextResolver creates a git context resolver.
func NewContextResolver() *ContextResolver {
	return &ContextResolver{}
}

func (r *ContextResolver) Name() string { return "git" }

func (r *ContextResolver) Resolve() ([]string, error) {
	branch := r.runGit("rev-parse", "--abbrev-ref", "HEAD")
	if branch == "" || branch == "HEAD" {
		return nil, nil // not a repo or detached HEAD
	}
	return []string{branch}, nil
}

func (p *ContentProvider) runGit(args ...string) string {
	return runGitCmd(p.dir, args...)
}

func (r *ContextResolver) runGit(args ...string) string {
	return runGitCmd(r.dir, args...)
}

func runGitCmd(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
```

**Step 4: Run tests**

Run: `go test ./internal/context/git/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/context/git/
git commit -m "feat: implement git content provider and context resolver"
```

---

### Task 11: Add provider/resolver registry

**Files:**
- Modify: `internal/context/context.go`

**Step 1: Write failing test**

Add to `internal/context/context_test.go`:

```go
func TestLookupContentProvider(t *testing.T) {
	p := LookupContentProvider("datetime")
	if p == nil {
		t.Fatal("expected datetime provider")
	}
	if p.Name() != "datetime" {
		t.Errorf("got name %q", p.Name())
	}
}

func TestLookupContentProvider_unknown(t *testing.T) {
	p := LookupContentProvider("nonexistent")
	if p != nil {
		t.Errorf("expected nil for unknown provider")
	}
}

func TestLookupContextResolver(t *testing.T) {
	r := LookupContextResolver("git")
	if r == nil {
		t.Fatal("expected git resolver")
	}
	if r.Name() != "git" {
		t.Errorf("got name %q", r.Name())
	}
}

func TestLookupContextResolver_unknown(t *testing.T) {
	r := LookupContextResolver("nonexistent")
	if r != nil {
		t.Errorf("expected nil for unknown resolver")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/context/ -run TestLookup -v`
Expected: FAIL

**Step 3: Implement**

Add to `internal/context/context.go`:

```go
import (
	"github.com/chris-regnier/diaryctl/internal/context/datetime"
	gitctx "github.com/chris-regnier/diaryctl/internal/context/git"
)

var contentProviders = map[string]func() ContentProvider{
	"datetime": func() ContentProvider { return datetime.New() },
	"git":      func() ContentProvider { return gitctx.NewContentProvider() },
}

var contextResolvers = map[string]func() ContextResolver{
	"git": func() ContextResolver { return gitctx.NewContextResolver() },
}

// LookupContentProvider returns a content provider by name, or nil if unknown.
func LookupContentProvider(name string) ContentProvider {
	factory, ok := contentProviders[name]
	if !ok {
		return nil
	}
	return factory()
}

// LookupContextResolver returns a context resolver by name, or nil if unknown.
func LookupContextResolver(name string) ContextResolver {
	factory, ok := contextResolvers[name]
	if !ok {
		return nil
	}
	return factory()
}
```

**Step 4: Run tests**

Run: `go test ./internal/context/ -run TestLookup -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/context/
git commit -m "feat: add content provider and context resolver registry"
```

---

### Task 12: Implement manual context state file

**Files:**
- Create: `internal/context/state.go`
- Create: `internal/context/state_test.go`

**Step 1: Write failing test**

Create `internal/context/state_test.go`:

```go
package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManualContexts_noFile(t *testing.T) {
	dir := t.TempDir()
	names, err := LoadManualContexts(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty, got %v", names)
	}
}

func TestSetAndLoadManualContexts(t *testing.T) {
	dir := t.TempDir()
	if err := SetManualContext(dir, "sprint:23"); err != nil {
		t.Fatalf("SetManualContext: %v", err)
	}
	if err := SetManualContext(dir, "project:auth"); err != nil {
		t.Fatalf("SetManualContext: %v", err)
	}
	names, err := LoadManualContexts(dir)
	if err != nil {
		t.Fatalf("LoadManualContexts: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2, got %d", len(names))
	}
}

func TestSetManualContext_idempotent(t *testing.T) {
	dir := t.TempDir()
	_ = SetManualContext(dir, "sprint:23")
	_ = SetManualContext(dir, "sprint:23")
	names, _ := LoadManualContexts(dir)
	if len(names) != 1 {
		t.Errorf("expected 1 after duplicate set, got %d", len(names))
	}
}

func TestUnsetManualContext(t *testing.T) {
	dir := t.TempDir()
	_ = SetManualContext(dir, "sprint:23")
	_ = SetManualContext(dir, "project:auth")
	if err := UnsetManualContext(dir, "sprint:23"); err != nil {
		t.Fatalf("UnsetManualContext: %v", err)
	}
	names, _ := LoadManualContexts(dir)
	if len(names) != 1 || names[0] != "project:auth" {
		t.Errorf("expected [project:auth], got %v", names)
	}
}

func TestUnsetManualContext_notSet(t *testing.T) {
	dir := t.TempDir()
	err := UnsetManualContext(dir, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/context/ -run TestLoadManual -v`
Expected: FAIL

**Step 3: Implement**

Create `internal/context/state.go`:

```go
package context

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const activeContextsFile = "active-contexts.json"

// LoadManualContexts reads the active manual context names from the state file.
func LoadManualContexts(dataDir string) ([]string, error) {
	path := filepath.Join(dataDir, activeContextsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return nil, err
	}
	return names, nil
}

// SetManualContext adds a context name to the active list (idempotent).
func SetManualContext(dataDir string, name string) error {
	names, err := LoadManualContexts(dataDir)
	if err != nil {
		return err
	}
	for _, n := range names {
		if n == name {
			return nil // already set
		}
	}
	names = append(names, name)
	return writeManualContexts(dataDir, names)
}

// UnsetManualContext removes a context name from the active list.
func UnsetManualContext(dataDir string, name string) error {
	names, err := LoadManualContexts(dataDir)
	if err != nil {
		return err
	}
	var filtered []string
	for _, n := range names {
		if n != name {
			filtered = append(filtered, n)
		}
	}
	return writeManualContexts(dataDir, filtered)
}

func writeManualContexts(dataDir string, names []string) error {
	if names == nil {
		names = []string{}
	}
	data, err := json.MarshalIndent(names, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dataDir, activeContextsFile)
	return os.WriteFile(path, data, 0644)
}
```

**Step 4: Run tests**

Run: `go test ./internal/context/ -run "Manual" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/context/state.go internal/context/state_test.go
git commit -m "feat: implement manual context state file (set/unset/load)"
```

---

### Task 13: Add context CLI subcommands

**Files:**
- Create: `cmd/context.go`

**Step 1: Implement all context subcommands**

Create `cmd/context.go` following the `cmd/template.go` pattern:

```go
package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/chris-regnier/diaryctl/internal/context"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage contexts",
	Long:  "Manage semantic contexts for grouping diary entries.",
}

var contextListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all contexts",
	Example: `  diaryctl context list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		contexts, err := store.ListContexts()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		if jsonOutput {
			ui.FormatJSON(os.Stdout, contexts)
		} else {
			ui.FormatContextList(os.Stdout, contexts)
		}
		return nil
	},
}

var contextShowCmd = &cobra.Command{
	Use:     "show <name>",
	Short:   "Show a context",
	Example: `  diaryctl context show feature/auth`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		ctx, err := store.GetContextByName(name)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "Error: context %q not found\n", name)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		if jsonOutput {
			ui.FormatJSON(os.Stdout, ctx)
		} else {
			ui.FormatContextFull(os.Stdout, ctx)
		}
		return nil
	},
}

var forceDeleteContext bool

var contextDeleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Short:   "Delete a context",
	Example: `  diaryctl context delete feature/auth`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		ctx, err := store.GetContextByName(name)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "Error: context %q not found\n", name)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		if !forceDeleteContext {
			fmt.Fprintf(os.Stdout, "Context: %s (%s)\n", ctx.Name, ctx.ID)
			confirmed, err := ui.Confirm("Delete this context? This cannot be undone.")
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(2)
			}
			if !confirmed {
				fmt.Fprintln(os.Stdout, "Cancelled.")
				return nil
			}
		}
		if err := store.DeleteContext(ctx.ID); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stdout, "Deleted context %q.\n", name)
		return nil
	},
}

var contextSetCmd = &cobra.Command{
	Use:     "set <name>",
	Short:   "Activate a manual context",
	Example: `  diaryctl context set sprint:23`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := context.SetManualContext(appConfig.DataDir, name); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stdout, "Activated context %q.\n", name)
		return nil
	},
}

var contextUnsetCmd = &cobra.Command{
	Use:     "unset <name>",
	Short:   "Deactivate a manual context",
	Example: `  diaryctl context unset sprint:23`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := context.UnsetManualContext(appConfig.DataDir, name); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stdout, "Deactivated context %q.\n", name)
		return nil
	},
}

var contextActiveCmd = &cobra.Command{
	Use:     "active",
	Short:   "Show currently active contexts",
	Example: `  diaryctl context active`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manual, err := context.LoadManualContexts(appConfig.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		var autoContexts []string
		for _, name := range appConfig.ContextResolvers {
			r := context.LookupContextResolver(name)
			if r == nil {
				continue
			}
			names, err := r.Resolve()
			if err != nil {
				continue
			}
			autoContexts = append(autoContexts, names...)
		}

		if jsonOutput {
			ui.FormatJSON(os.Stdout, map[string][]string{
				"manual": manual,
				"auto":   autoContexts,
			})
		} else {
			ui.FormatActiveContexts(os.Stdout, manual, autoContexts)
		}
		return nil
	},
}

func init() {
	contextDeleteCmd.Flags().BoolVar(&forceDeleteContext, "force", false, "skip confirmation prompt")

	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextShowCmd)
	contextCmd.AddCommand(contextDeleteCmd)
	contextCmd.AddCommand(contextSetCmd)
	contextCmd.AddCommand(contextUnsetCmd)
	contextCmd.AddCommand(contextActiveCmd)

	rootCmd.AddCommand(contextCmd)
}
```

**Step 2: Verify compilation (will fail — needs ui formatting functions)**

Run: `go build ./cmd/`
Expected: FAIL — missing ui formatting functions

**Step 3: Commit (partial — UI functions in next task)**

```bash
git add cmd/context.go
git commit -m "feat: add context CLI subcommands (list, show, delete, set, unset, active)"
```

---

### Task 14: Add context formatting to ui/output.go

**Files:**
- Modify: `internal/ui/output.go`

**Step 1: Add FormatContextList**

```go
// FormatContextList prints a table of contexts.
func FormatContextList(w io.Writer, contexts []storage.Context) {
	if len(contexts) == 0 {
		fmt.Fprintln(w, "No contexts found.")
		return
	}
	for _, c := range contexts {
		fmt.Fprintf(w, "%s  %s  %s  %s\n", c.Name, c.Source, c.ID, c.UpdatedAt.Local().Format("2006-01-02 15:04"))
	}
}
```

**Step 2: Add FormatContextFull**

```go
// FormatContextFull prints full details of a context.
func FormatContextFull(w io.Writer, c storage.Context) {
	fmt.Fprintf(w, "Context: %s\n", c.Name)
	fmt.Fprintf(w, "ID: %s\n", c.ID)
	fmt.Fprintf(w, "Source: %s\n", c.Source)
	fmt.Fprintf(w, "Created: %s\n", c.CreatedAt.Local().Format("2006-01-02 15:04"))
	fmt.Fprintf(w, "Modified: %s\n", c.UpdatedAt.Local().Format("2006-01-02 15:04"))
}
```

**Step 3: Add FormatActiveContexts**

```go
// FormatActiveContexts prints the currently active contexts.
func FormatActiveContexts(w io.Writer, manual []string, auto []string) {
	if len(manual) == 0 && len(auto) == 0 {
		fmt.Fprintln(w, "No active contexts.")
		return
	}
	if len(manual) > 0 {
		fmt.Fprintf(w, "manual:  %s\n", strings.Join(manual, ", "))
	}
	if len(auto) > 0 {
		fmt.Fprintf(w, "auto:    %s\n", strings.Join(auto, ", "))
	}
}
```

**Step 4: Update FormatEntryFull to show contexts**

After the templates display in `FormatEntryFull`, add:

```go
	if len(e.Contexts) > 0 {
		names := make([]string, len(e.Contexts))
		for i, ref := range e.Contexts {
			names[i] = ref.ContextName
		}
		fmt.Fprintf(w, "Contexts: %s\n", strings.Join(names, ", "))
	}
```

**Step 5: Verify compilation**

Run: `go build ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/ui/output.go
git commit -m "feat: add context formatting to ui output"
```

---

### Task 15: Integrate context attachment into create command

**Files:**
- Modify: `cmd/create.go`

**Step 1: Add context resolution after entry creation**

After the entry is created and stored, add context attachment:

```go
	// Resolve and attach contexts
	var providers []context.ContentProvider
	for _, name := range appConfig.ContextProviders {
		p := context.LookupContentProvider(name)
		if p != nil {
			providers = append(providers, p)
		}
	}

	var resolvers []context.ContextResolver
	for _, name := range appConfig.ContextResolvers {
		r := context.LookupContextResolver(name)
		if r != nil {
			resolvers = append(resolvers, r)
		}
	}

	manualContexts, _ := context.LoadManualContexts(appConfig.DataDir)
	ctxRefs, err := context.ResolveActiveContexts(resolvers, manualContexts, store)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: resolving contexts: %v\n", err)
	}
	for _, ref := range ctxRefs {
		if attachErr := store.AttachContext(e.ID, ref.ContextID); attachErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: attaching context %q: %v\n", ref.ContextName, attachErr)
		}
	}
```

Also add content provider composition before the editor opens (when building editor buffer):

```go
	// Compose content from providers + template
	if templateContent != "" || len(providers) > 0 {
		content = context.ComposeContent(providers, templateContent)
	}
```

**Step 2: Add import**

```go
	"github.com/chris-regnier/diaryctl/internal/context"
```

**Step 3: Verify compilation**

Run: `go build ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/create.go
git commit -m "feat: integrate context attachment and content composition into create command"
```

---

### Task 16: Integrate context attachment into jot command

**Files:**
- Modify: `cmd/jot.go`

**Step 1: Add context resolution after jot**

After the entry is created/updated via jot, add context attachment (same pattern as create but without content providers, since jot doesn't open an editor):

```go
	// Resolve and attach contexts
	var resolvers []context.ContextResolver
	for _, name := range appConfig.ContextResolvers {
		r := context.LookupContextResolver(name)
		if r != nil {
			resolvers = append(resolvers, r)
		}
	}
	manualContexts, _ := context.LoadManualContexts(appConfig.DataDir)
	ctxRefs, _ := context.ResolveActiveContexts(resolvers, manualContexts, store)
	for _, ref := range ctxRefs {
		_ = store.AttachContext(e.ID, ref.ContextID)
	}
```

**Step 2: Verify compilation**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/jot.go
git commit -m "feat: integrate context attachment into jot command"
```

---

### Task 17: Integrate context attachment into today command

**Files:**
- Modify: `cmd/today.go`

**Step 1: Add context resolution**

Same pattern as jot — after `GetOrCreateToday`, attach contexts:

```go
	// Resolve and attach contexts
	var resolvers []context.ContextResolver
	for _, name := range appConfig.ContextResolvers {
		r := context.LookupContextResolver(name)
		if r != nil {
			resolvers = append(resolvers, r)
		}
	}
	manualContexts, _ := context.LoadManualContexts(appConfig.DataDir)
	ctxRefs, _ := context.ResolveActiveContexts(resolvers, manualContexts, store)
	for _, ref := range ctxRefs {
		_ = store.AttachContext(e.ID, ref.ContextID)
	}
```

**Step 2: Verify compilation**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/today.go
git commit -m "feat: integrate context attachment into today command"
```

---

### Task 18: Add --context filter to list and show commands

**Files:**
- Modify: `cmd/list.go`
- Modify: `cmd/show.go`

**Step 1: Add --context flag to list**

Add flag variable:

```go
var listContextFilter string
```

Add to `init()`:

```go
listCmd.Flags().StringVar(&listContextFilter, "context", "", "filter by context name")
```

Pass to ListOptions:

```go
opts := storage.ListOptions{
	// ... existing fields ...
	ContextName: listContextFilter,
}
```

**Step 2: Update list entry display to show context labels**

In the list output formatting, show context labels if present. Update `FormatEntryList` in `internal/ui/output.go` (or wherever list entries are printed) to include context names:

```go
if len(e.Contexts) > 0 {
	names := make([]string, len(e.Contexts))
	for i, ref := range e.Contexts {
		names[i] = ref.ContextName
	}
	fmt.Fprintf(w, "  [%s]", strings.Join(names, ", "))
}
```

**Step 3: Verify compilation**

Run: `go build ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/list.go cmd/show.go internal/ui/output.go
git commit -m "feat: add --context filter to list command and context labels to output"
```

---

### Task 19: Run full test suite and fix issues

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: PASS (fix any compilation or test failures)

**Step 2: Run vet and build**

Run: `go vet ./... && go build ./...`
Expected: PASS

**Step 3: Commit any fixes**

```bash
git add -A
git commit -m "fix: resolve test and compilation issues from context integration"
```

---

### Task 20: Manual integration test

**Step 1: Test manual context workflow**

```bash
go run . context set sprint:23
go run . context active
go run . context list
go run . jot "test with context"
go run . today
go run . list --context sprint:23
go run . context unset sprint:23
go run . context active
```

Verify each command works correctly. Fix any issues found.

**Step 2: Final commit**

```bash
git add -A
git commit -m "fix: integration test fixes for context providers"
```
