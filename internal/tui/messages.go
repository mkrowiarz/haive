package tui

type projectLoadedMsg struct {
	name   string
	ptype  string
	status string
}

type worktreesLoadedMsg struct {
	worktrees []worktreeInfo
}

type worktreeInfo struct {
	branch string
	path   string
	isMain bool
}

type databasesLoadedMsg struct {
	databases []databaseInfo
}

type databaseInfo struct {
	name      string
	isDefault bool
}

type dumpsLoadedMsg struct {
	dumps []dumpInfo
}

type dumpInfo struct {
	name string
	size string
	date string
}

type dumpFinishedMsg struct {
	result *dumpResult
	err    error
}

type dumpResult struct {
	path string
}

type cloneFinishedMsg struct {
	result *cloneResult
	err    error
}

type cloneResult struct {
	targetDB string
}

type dropFinishedMsg struct {
	result *dropResult
	err    error
}

type dropResult struct {
	dbName string
}

type importFinishedMsg struct {
	result *importResult
	err    error
}

type importResult struct {
	dbName string
}

type worktreeCreatedMsg struct {
	result *worktreeResult
	err    error
}

type worktreeResult struct {
	branch string
	path   string
}

type worktreeRemovedMsg struct {
	result *worktreeResult
	err    error
}
