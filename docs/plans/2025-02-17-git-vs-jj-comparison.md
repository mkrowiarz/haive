# Git Worktrees vs Jujutsu (jj): VCS Strategy Comparison for Haive

> Context: Symfony projects with multiple concurrent development instances

## Your Scenario

- **Framework**: Symfony (PHP)
- **Goal**: Run multiple feature branches simultaneously
- **Shared resources**: Database (main DB shared, per-branch DBs optional)
- **Isolated resources**: `vendor/`, `node_modules/`, `var/cache/`
- **Use case**: AI agent coding in isolated branches while keeping main stable

---

## Approach 1: Git Worktrees (Current Plan)

### How It Works

```
~/projects/symfony-app/              # main worktree (main branch)
├── .git/
├── vendor/                          # main dependencies
├── node_modules/                    # main JS deps
├── var/cache/
├── compose.yaml                     # main docker-compose
└── .worktrees/                      # linked worktrees
    ├── feature-auth/                # feature/auth branch
    │   ├── vendor/                  # isolated (bind mount)
    │   ├── node_modules/            # isolated
    │   ├── compose.worktree.yaml    # per-worktree docker override
    │   └── .env.local               # copied from main
    └── feature-api/                 # feature/api branch
        ├── vendor/
        ├── node_modules/
        └── compose.worktree.yaml
```

### Daily Workflow

```bash
# Start working on new feature
cd ~/projects/symfony-app
haive worktree create feature/new-api

# Creates:
# - .worktrees/feature-new-api/
# - Clones DB: main_db → main_db_feature_new_api
# - Copies .env.local, updates DATABASE_URL
# - Runs: composer install, npm ci

# Work on it
cd .worktrees/feature-new-api
# Edit code...

# Start isolated container
haive serve
# Container runs with isolated vendor/, connected to shared DB service

# Switch back to main
cd ~/projects/symfony-app
# main is untouched, stable

# Cleanup when done
cd ~/projects/symfony-app
haive worktree remove feature/new-api
# Drops DB, removes directory
```

### Pros

| Advantage | Explanation |
|-----------|-------------|
| **True isolation** | Each worktree is a separate directory with own files |
| **Independent dependencies** | Different Symfony versions per branch possible |
| **Familiar model** | Standard Git worktrees - well documented |
| **IDE friendly** | Each worktree is a separate project root |
| **Concurrent containers** | Each worktree can run its own Docker container |
| **Easy cleanup** | Delete directory = gone |

### Cons

| Disadvantage | Explanation |
|--------------|-------------|
| **Directory juggling** | Must `cd` between worktrees |
| **Disk usage** | Multiple copies of `vendor/`, `node_modules/` |
| **Config duplication** | Need `compose.worktree.yaml` per worktree |
| **Complex paths** | Relative paths can get confusing |
| **Tooling friction** | Some tools expect single git root |

### Isolation Mechanism (Docker)

```yaml
# compose.worktree.yaml in each worktree
services:
  app:
    volumes:
      - .:/app:delegated
      - /app/vendor          # Named volume = isolated
      - /app/var/cache       # Named volume = isolated
      - /app/node_modules    # Named volume = isolated

networks:
  app-network:
    external: true  # Connect to main's DB, redis, etc.
```

---

## Approach 2: Jujutsu (jj)

### How It Works

```
~/projects/symfony-app/              # single directory
├── .git/                            # underlying git repo
├── .jj/                             # jj state
├── vendor/                          # CURRENT change's deps
├── node_modules/                    # CURRENT change's JS deps
├── var/cache/                       # CURRENT change's cache
├── compose.yaml                     # can be templated per change
└── .env.local                       # CURRENT change's env

# jj changes (not directories):
# - qpvqkxkz: main (stable)
# - vvmkmwwo: feature/auth  
# - xpzkwrlk: feature/api
```

### Daily Workflow

```bash
# Start working on new feature
cd ~/projects/symfony-app
jj new -m "feat: new api endpoint"

# jj automatically switches working copy to new change
# But: vendor/, node_modules/, .env.local stay as they were!

# Haive would need to handle the switch:
haive jj start --clone-from=main

# This would:
# 1. Clone DB: main_db → main_db_feature_new_api
# 2. Update .env.local with new DATABASE_URL
# 3. Run: composer install, npm ci (potentially different versions!)
# 4. Set git config: haive.database = main_db_feature_new_api

# Work on it
# Edit code...

# Start container (same directory, different DB)
haive serve
# But wait: how do we run two containers from same directory?

# Switch to another change
jj edit vvmkmwwo  # switch to feature/auth

# Haive detects change switch:
haive jj switch

# This would:
# 1. Stop current container
# 2. Update .env.local to feature/auth DB
# 3. Run composer install (different versions possible!)
# 4. Start container for this change

# Go back to main
jj edit qpvqkxkz
haive jj switch

# Cleanup
jj abandon vvmkmwwo
haive jj cleanup  # drop associated DB
```

### Pros

| Advantage | Explanation |
|-----------|-------------|
| **No directory juggling** | Always in same directory |
| **Simpler mental model** | One project, multiple changes |
| **Fast switching** | `jj edit` is instant (no `cd`) |
| **Better history** | jj's changelog is superior to git |
| **Colocated git** | Works with GitHub, existing tools |
| **Conflict resolution** | jj's 3-way merge is excellent |

### Cons

| Disadvantage | Explanation |
|--------------|-------------|
| **No concurrent instances** | Only ONE change active at a time |
| **Dependency switching cost** | `composer install` on every switch if versions differ |
| **Single IDE instance** | Can't have two branches open side-by-side in different windows |
| **Container restart** | Must stop/start container on every change switch |
| **Cache invalidation** | Symfony cache cleared on every switch |
| **Complex for AI agents** | Agent must handle change switching, not just work in directory |

### The Isolation Problem

**Critical question**: Can jj achieve the same dependency isolation as worktrees?

**Option A: Re-install on every switch** ❌ Slow
```bash
jj edit feature-x
composer install  # 30-60 seconds
npm ci            # 30-60 seconds
# Finally ready to work
```

**Option B: Use Docker volumes for isolation** ⚠️ Complex
```yaml
# compose.yaml with change-specific volumes
services:
  app:
    volumes:
      - .:/app:delegated
      - jj-vendor-${JJ_CHANGE_ID}:/app/vendor
      - jj-node-${JJ_CHANGE_ID}:/app/node_modules
```
This requires:
- Pass change ID to Docker
- Manage volume lifecycle
- Cleanup when change abandoned

**Option C: Use `jj workspace` (experimental)** ⚠️ Bleeding edge
```bash
jj workspace create ../feature-x
# Creates separate working directory
```
This is essentially re-implementing git worktrees within jj.

---

## Side-by-Side Comparison

| Aspect | Git Worktrees | Jujutsu (jj) |
|--------|---------------|--------------|
| **Concurrent instances** | ✅ Yes - multiple directories | ❌ No - single directory |
| **Dependency isolation** | ✅ Natural (separate directories) | ⚠️ Requires Docker volumes or reinstall |
| **Switch time** | ⚠️ `cd` + context switch | ✅ Instant `jj edit` |
| **Disk usage** | ❌ N copies of vendor/ | ✅ Single copy (or managed volumes) |
| **IDE support** | ✅ Multiple project windows | ❌ Single project window |
| **AI agent friendly** | ✅ Agent works in directory | ⚠️ Agent must manage change state |
| **Container model** | ✅ One container per worktree | ⚠️ Stop/start container on switch |
| **Cleanup** | ✅ Delete directory | ✅ `jj abandon` |
| **Symfony cache** | ✅ Persistent per worktree | ❌ Cleared on switch |
| **Learning curve** | ✅ Standard Git | ⚠️ New tool |
| **GitHub integration** | ✅ Native | ✅ Via colocated git |

---

## Recommendation for Your Use Case

### Stick with Git Worktrees IF:

1. **You need concurrent running instances**
   - Testing API in branch A while working on UI in branch B
   - Comparing behavior side-by-side

2. **Different dependency versions per branch**
   - Branch A: Symfony 6.4
   - Branch B: Symfony 7.0 (experimental)

3. **AI agents work independently**
   - Agent 1 works in `.worktrees/feature-a/`
   - Agent 2 works in `.worktrees/feature-b/`
   - No coordination needed

4. **IDE workflow**
   - Open `.worktrees/feature-a/` in VS Code window 1
   - Open `.worktrees/feature-b/` in VS Code window 2

### Consider Jujutsu IF:

1. **Sequential work only**
   - Work on A, finish, switch to B
   - Never need A and B simultaneously

2. **Fast context switching priority**
   - Code review: switch, check, switch back
   - Quick fixes across branches

3. **History/change management matters**
   - Heavy use of `jj squash`, `jj split`
   - Complex history rewriting

4. **Disk space constrained**
   - vendor/ is huge (1GB+)
   - Can't afford N copies

---

## Hybrid Approach (Future Idea)

Could Haive support both?

```toml
[vcs]
type = "auto"  # detect git vs jj

[worktree]
# For git: use traditional worktrees
# For jj: use workspaces (when stable) or Docker volumes
strategy = "auto"
```

**Git mode:** Current plan - separate directories, full isolation

**jj mode:** 
- Use `jj workspace` (when stable) OR
- Use Docker volume isolation:
  ```yaml
  volumes:
    - jj-vendor-${BRANCH}:/app/vendor
  ```
- Haive manages:
  - DB cloning
  - .env.local switching
  - Container lifecycle per change

---

## Concrete Example: Your Symfony Setup

### Git Worktree Version

```bash
# Terminal 1: Main project
cd ~/projects/symfony-app
symfony server:start
# http://localhost:8000 - stable main

# Terminal 2: Feature A
cd ~/projects/symfony-app/.worktrees/feature-a
symfony server:start --port=8001
# http://localhost:8001 - feature A
# Uses: main_db_feature_a

# Terminal 3: Feature B  
cd ~/projects/symfony-app/.worktrees/feature-b
symfony server:start --port=8002
# http://localhost:8002 - feature B
# Uses: main_db_feature_b

# All three run simultaneously with:
# - Different code (branches)
# - Different vendor/ (isolated)
# - Different databases
# - Shared services (DB container, Redis)
```

### Jujutsu Version

```bash
# Terminal 1: Main (stable)
cd ~/projects/symfony-app
jj edit main
symfony server:start
# http://localhost:8000

# Switch to Feature A
jj edit feature-a
haive jj switch  # updates .env.local, runs composer install
symfony server:start
# http://localhost:8000 (same port, different code)
# Uses: main_db_feature_a

# CANNOT have Feature B running simultaneously
# Must stop, switch, restart:
jj edit feature-b
haive jj switch
symfony server:start
# http://localhost:8000
# Uses: main_db_feature_b
```

---

## Verdict

**For your use case (Symfony, AI agents, concurrent instances, shared DB + isolated dependencies):**

→ **Git Worktrees are the right choice for Phase 1**

Jujutsu support could be added later as an alternative VCS backend, but it requires solving the isolation problem (Docker volumes or `jj workspace`).

The directory-based isolation of worktrees maps perfectly to:
- Separate Docker containers per branch
- Independent vendor/node_modules
- Concurrent running instances
- Simple AI agent workflow (just work in this directory)
