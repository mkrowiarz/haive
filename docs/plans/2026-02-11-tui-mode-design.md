# TUI Mode - Design Document

## Overview

Interactive terminal interface for managing Docker Compose-based development projects. Keyboard-driven with visible shortcuts, inspired by lazygit/lazydocker.

## Layout

**Master-detail split:**
- **Left column (narrow):** 3 stacked sections
  - Project Info (compact)
  - Worktrees (list)
  - Dumps (list)
- **Right column (wide):** Databases pane (main)
- **Bottom:** Status bar with shortcuts

```
┌─────────────┬────────────────────────────────────────┐
│Info         │Databases                              │
│─────────────│───────────────────────────────────────│
│phoenix      │ mytower_eu              (default)     │
│symfony      │ mytower_eu_test                       │
│             │ mytower_eu_wt_abc                     │
├─────────────┤                                       │
│Worktrees    │                                       │
│─────────────│ Selected: mytower_eu                 │
│*main        │ Size: 245 MB                          │
│feat/abc     │ Tables: 47                            │
│feat/xyz     │                                       │
├─────────────┤                                       │
│Dumps        │                                       │
│─────────────│                                       │
│eu_02.sql    │                                       │
│eu_01.sql    │                                       │
└─────────────┴───────────────────────────────────────┘
[d]ump [c]lone [i]mport [n]ew worktree [r]efresh [q]uit
```

## Navigation

- `Tab` / `Shift+Tab` - Cycle between panes
- `1` `2` `3` `4` - Jump directly to pane
- `↑` `↓` - Navigate within pane
- `Enter` - Show action menu (databases only)

## Panes

### Project Info (Pane 1)
- Compact, non-interactive
- Shows: Project name, type, docker compose status

### Worktrees (Pane 2)
- List format: `* branch_name` (asterisk = current)
- Actions:
  - `n` - New worktree (prompt for branch name)
  - `r` - Remove selected (y/n confirm)
  - `o` - Open in terminal
- No Enter action

### Databases (Pane 3 - main)
- List format: `database_name | 245 MB`
- Asterisk on default DB
- Selected shows detail at bottom (size, tables)
- `Enter` - Dropdown menu: Dump / Clone / Drop / View dumps
- Actions:
  - `d` - Dump selected
  - `c` - Clone (prompt for target name)
  - `x` - Drop (y/n confirm)

### Dumps (Pane 4)
- List format: `filename | 12 MB | 2024-02-11`
- Actions:
  - `i` - Import (select target DB from list)
  - `x` - Delete dump file (y/n confirm)
- No Enter action

## Operations & Feedback

### Long-running operations
- Modal overlay blocks UI
- Shows: operation name, spinner, elapsed time
- On success: brief "✓ Done" message, auto-dismiss
- On error: error modal

### Error handling
- Full modal overlay with error message
- "Press any key to dismiss"

### Confirmations
- Simple y/n prompt in status bar area
- `y` executes, `n` or `Esc` cancels

### Auto-refresh
- After successful action, relevant pane refreshes
- `r` - Refresh current pane
- `R` - Refresh all panes

## Key Bindings

### Global
| Key | Action |
|-----|--------|
| `1-4` | Jump to pane |
| `Tab` | Next pane |
| `Shift+Tab` | Previous pane |
| `r` | Refresh current pane |
| `R` | Refresh all |
| `q` | Quit |
| `?` | Help overlay |

### Databases
| Key | Action |
|-----|--------|
| `d` | Dump |
| `c` | Clone (prompt) |
| `x` | Drop (confirm) |
| `Enter` | Action menu |

### Worktrees
| Key | Action |
|-----|--------|
| `n` | New (prompt) |
| `r` | Remove (confirm) |
| `o` | Open in terminal |

### Dumps
| Key | Action |
|-----|--------|
| `i` | Import (select DB) |
| `x` | Delete (confirm) |

## Technical Architecture

### Entry Point
- `./pm` (no flags) → TUI
- `./pm --mcp` → MCP server
- `./pm <command>` → CLI (future)

### Package Structure
```
internal/tui/
├── app.go          # Main Bubble Tea app
├── styles.go       # Lip Gloss styles
├── components/
│   ├── infopane.go
│   ├── worktrees.go
│   ├── databases.go
│   ├── dumps.go
│   ├── statusbar.go
│   ├── modal.go
│   └── menu.go
└── messages.go
```

### Dependencies
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles` - Components (spinner, etc.)

## Implementation Phases

### Phase 1: Foundation
- Bubble Tea setup
- Basic layout
- Tab/number navigation
- Empty panes with labels

### Phase 2: Read-only Panes
- Project Info display
- Worktrees list
- Databases list
- Dumps list
- Status bar

### Phase 3: Actions
- Dump/Clone/Import with progress
- Worktree create/remove
- Error modal
- Confirmations
- Auto-refresh

### Phase 4: Polish
- Dropdown action menu
- Help overlay
- Edge cases
