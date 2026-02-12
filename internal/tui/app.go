package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/commands"
)

type inputMode string

const (
	inputModeNone   inputMode = ""
	inputModeClone  inputMode = "clone"
	inputModeNewWT  inputMode = "newwt"
	inputModeImport inputMode = "import"
)

type confirmMode string

const (
	confirmModeNone   confirmMode = ""
	confirmModeDrop   confirmMode = "drop"
	confirmModeRemove confirmMode = "remove"
)

type Model struct {
	focusedPane   int
	width         int
	height        int
	projectRoot   string
	projectName   string
	projectType   string
	projectStatus string
	worktrees     []worktreeInfo
	databases     []databaseInfo
	dumps         []dumpInfo
	selectedIndex map[int]int
	scrollOffset  map[int]int

	showModal    bool
	modalTitle   string
	modalMessage string
	showError    bool
	errorMessage string
	showHelp     bool

	inputMode   inputMode
	inputValue  string
	inputPrompt string

	confirmMode   confirmMode
	confirmTarget string
	confirmPath   string

	importDumpName string
}

const maxVisibleLines = 6

func NewModel() Model {
	return Model{
		focusedPane:   3,
		projectRoot:   ".",
		projectName:   "Loading...",
		selectedIndex: map[int]int{1: 0, 2: 0, 3: 0, 4: 0},
		scrollOffset:  map[int]int{1: 0, 2: 0, 3: 0, 4: 0},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadProject, m.loadWorktrees, m.loadDatabases, m.loadDumps)
}

func (m Model) loadProject() tea.Msg {
	info, err := commands.Info(m.projectRoot)
	if err != nil {
		return projectLoadedMsg{name: "Error", ptype: err.Error(), status: "✗"}
	}

	name := "Unknown"
	ptype := "generic"
	status := "✗"

	if info.ConfigSummary != nil {
		name = info.ConfigSummary.Name
		ptype = info.ConfigSummary.Type
	}
	if info.DockerComposeExists {
		status = "✓"
	}

	return projectLoadedMsg{name: name, ptype: ptype, status: status}
}

func (m Model) loadWorktrees() tea.Msg {
	result, err := commands.List(m.projectRoot)
	if err != nil {
		return worktreesLoadedMsg{worktrees: []worktreeInfo{}}
	}

	var wtis []worktreeInfo
	for _, wt := range result {
		wtis = append(wtis, worktreeInfo{
			branch: wt.Branch,
			path:   wt.Path,
			isMain: wt.IsMain,
		})
	}
	return worktreesLoadedMsg{worktrees: wtis}
}

func (m Model) loadDatabases() tea.Msg {
	result, err := commands.ListDBs(m.projectRoot)
	if err != nil {
		return databasesLoadedMsg{databases: []databaseInfo{}}
	}

	var dbis []databaseInfo
	for _, db := range result.Databases {
		dbis = append(dbis, databaseInfo{
			name:      db.Name,
			isDefault: db.IsDefault,
		})
	}
	return databasesLoadedMsg{databases: dbis}
}

func (m Model) loadDumps() tea.Msg {
	result, err := commands.ListDumps(m.projectRoot)
	if err != nil {
		return dumpsLoadedMsg{dumps: []dumpInfo{}}
	}

	var dis []dumpInfo
	for _, d := range result.Dumps {
		dis = append(dis, dumpInfo{
			name: d.Name,
			size: formatSize(d.Size),
			date: formatDate(d.Modified),
		})
	}
	return dumpsLoadedMsg{dumps: dis}
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDate(modified string) string {
	t, err := time.Parse(time.RFC3339, modified)
	if err != nil {
		return modified
	}
	return t.Format("Jan 02 15:04")
}

func shortenFilename(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}

	keepLen := (maxLen - 3) / 2
	return name[:keepLen] + "..." + name[len(name)-keepLen:]
}

func (m Model) refreshCurrentPane() tea.Cmd {
	switch m.focusedPane {
	case 1:
		return m.loadProject
	case 2:
		return m.loadWorktrees
	case 3:
		return m.loadDatabases
	case 4:
		return m.loadDumps
	}
	return nil
}

func (m Model) refreshAll() tea.Cmd {
	return tea.Batch(m.loadProject, m.loadWorktrees, m.loadDatabases, m.loadDumps)
}

func (m Model) statusBarText() string {
	if m.inputMode != inputModeNone {
		return m.inputPrompt + m.inputValue + "_ [Enter]confirm [Esc]cancel"
	}
	if m.confirmMode != confirmModeNone {
		return "Confirm " + string(m.confirmMode) + "? [y]es [n]o"
	}
	switch m.focusedPane {
	case 2:
		return "[n]ew [x]remove [o]pen [?]help [q]uit"
	case 3:
		return "[d]ump [c]lone [x]drop [?]help [q]uit"
	case 4:
		return "[i]mport [x]delete [?]help [q]uit"
	default:
		return "[Tab]switch [R]efresh all [?]help [q]uit"
	}
}

func (m Model) dumpDatabase(dbName string) tea.Cmd {
	return func() tea.Msg {
		result, err := commands.Dump(m.projectRoot, dbName, nil)
		if err != nil {
			return dumpFinishedMsg{err: err}
		}
		return dumpFinishedMsg{result: &dumpResult{path: result.Path}}
	}
}

func (m Model) cloneDatabase(sourceDB, targetDB string) tea.Cmd {
	return func() tea.Msg {
		result, err := commands.CloneDB(m.projectRoot, sourceDB, targetDB)
		if err != nil {
			return cloneFinishedMsg{err: err}
		}
		return cloneFinishedMsg{result: &cloneResult{targetDB: result.Target}}
	}
}

func (m Model) dropDatabase(dbName string) tea.Cmd {
	return func() tea.Msg {
		result, err := commands.DropDB(m.projectRoot, dbName)
		if err != nil {
			return dropFinishedMsg{err: err}
		}
		return dropFinishedMsg{result: &dropResult{dbName: result.Database}}
	}
}

func (m Model) importDump(dbName, dumpName string) tea.Cmd {
	return func() tea.Msg {
		result, err := commands.ImportDB(m.projectRoot, dbName, dumpName)
		if err != nil {
			return importFinishedMsg{err: err}
		}
		return importFinishedMsg{result: &importResult{dbName: result.Database}}
	}
}

func (m Model) createWorktree(branch string) tea.Cmd {
	return func() tea.Msg {
		result, err := commands.Create(m.projectRoot, branch, true)
		if err != nil {
			return worktreeCreatedMsg{err: err}
		}
		return worktreeCreatedMsg{result: &worktreeResult{branch: result.Branch, path: result.Path}}
	}
}

func (m Model) removeWorktree(branch string) tea.Cmd {
	return func() tea.Msg {
		result, err := commands.Remove(m.projectRoot, branch)
		if err != nil {
			return worktreeRemovedMsg{err: err}
		}
		return worktreeRemovedMsg{result: &worktreeResult{path: result.Path}}
	}
}

func (m Model) openInTerminal(path string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", "-a", "Terminal", path)
		case "linux":
			cmd = exec.Command("x-terminal-emulator", "--working-directory", path)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", "cmd", "/K", "cd", path)
		default:
			return nil
		}
		_ = cmd.Start()
		return nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case projectLoadedMsg:
		m.projectName = msg.name
		m.projectType = msg.ptype
		m.projectStatus = msg.status

	case worktreesLoadedMsg:
		m.worktrees = msg.worktrees

	case databasesLoadedMsg:
		m.databases = msg.databases

	case dumpsLoadedMsg:
		m.dumps = msg.dumps

	case dumpFinishedMsg:
		m.showModal = false
		if msg.err != nil {
			m.showError = true
			m.errorMessage = fmt.Sprintf("Dump failed: %v", msg.err)
		} else {
			m.showModal = true
			m.modalTitle = "✓ Dump Complete"
			m.modalMessage = fmt.Sprintf("Created: %s", msg.result.path)
		}
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return hideModalMsg{}
		})

	case cloneFinishedMsg:
		m.showModal = false
		if msg.err != nil {
			m.showError = true
			m.errorMessage = fmt.Sprintf("Clone failed: %v", msg.err)
		} else {
			m.showModal = true
			m.modalTitle = "✓ Clone Complete"
			m.modalMessage = fmt.Sprintf("Created: %s", msg.result.targetDB)
		}
		return m, tea.Batch(m.loadDatabases, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return hideModalMsg{}
		}))

	case dropFinishedMsg:
		m.showModal = false
		m.confirmMode = confirmModeNone
		if msg.err != nil {
			m.showError = true
			m.errorMessage = fmt.Sprintf("Drop failed: %v", msg.err)
		} else {
			m.showModal = true
			m.modalTitle = "✓ Database Dropped"
			m.modalMessage = msg.result.dbName
		}
		return m, tea.Batch(m.loadDatabases, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return hideModalMsg{}
		}))

	case importFinishedMsg:
		m.showModal = false
		m.inputMode = inputModeNone
		if msg.err != nil {
			m.showError = true
			m.errorMessage = fmt.Sprintf("Import failed: %v", msg.err)
		} else {
			m.showModal = true
			m.modalTitle = "✓ Import Complete"
			m.modalMessage = fmt.Sprintf("Imported to: %s", msg.result.dbName)
		}
		return m, tea.Batch(m.loadDatabases, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return hideModalMsg{}
		}))

	case worktreeCreatedMsg:
		m.showModal = false
		m.inputMode = inputModeNone
		if msg.err != nil {
			m.showError = true
			m.errorMessage = fmt.Sprintf("Create failed: %v", msg.err)
		} else {
			m.showModal = true
			m.modalTitle = "✓ Worktree Created"
			m.modalMessage = fmt.Sprintf("%s at %s", msg.result.branch, msg.result.path)
		}
		return m, tea.Batch(m.loadWorktrees, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return hideModalMsg{}
		}))

	case worktreeRemovedMsg:
		m.showModal = false
		m.confirmMode = confirmModeNone
		if msg.err != nil {
			m.showError = true
			m.errorMessage = fmt.Sprintf("Remove failed: %v", msg.err)
		} else {
			m.showModal = true
			m.modalTitle = "✓ Worktree Removed"
			m.modalMessage = msg.result.path
		}
		return m, tea.Batch(m.loadWorktrees, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return hideModalMsg{}
		}))

	case hideModalMsg:
		m.showModal = false
		m.showError = false

	case tea.KeyMsg:
		if m.showError {
			m.showError = false
			return m, nil
		}

		if m.showHelp {
			if msg.String() == "?" || msg.String() == "esc" || msg.String() == "q" {
				m.showHelp = false
			}
			return m, nil
		}

		if m.inputMode != inputModeNone {
			return m.handleInputMode(msg)
		}

		if m.confirmMode != confirmModeNone {
			return m.handleConfirmMode(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?":
			m.showHelp = true
		case "tab":
			m.focusedPane = m.focusedPane%4 + 1
		case "1", "2", "3", "4":
			m.focusedPane = int(msg.String()[0] - '0')
		case "r":
			return m, m.refreshCurrentPane()
		case "R":
			return m, m.refreshAll()
		case "up", "k":
			if idx, ok := m.selectedIndex[m.focusedPane]; ok && idx > 0 {
				m.selectedIndex[m.focusedPane] = idx - 1
				m.clampScrollOffset(m.focusedPane)
			}
		case "down", "j":
			idx := m.selectedIndex[m.focusedPane]
			maxIdx := 0
			switch m.focusedPane {
			case 2:
				maxIdx = len(m.worktrees) - 1
			case 3:
				maxIdx = len(m.databases) - 1
			case 4:
				maxIdx = len(m.dumps) - 1
			}
			if idx < maxIdx {
				m.selectedIndex[m.focusedPane] = idx + 1
				m.clampScrollOffset(m.focusedPane)
			}
		case "d":
			if m.focusedPane == 3 && len(m.databases) > 0 {
				idx := m.selectedIndex[3]
				if idx < len(m.databases) {
					db := m.databases[idx]
					m.showModal = true
					m.modalTitle = "Dumping database..."
					m.modalMessage = db.name
					return m, m.dumpDatabase(db.name)
				}
			}
		case "c":
			if m.focusedPane == 3 && len(m.databases) > 0 {
				idx := m.selectedIndex[3]
				if idx < len(m.databases) {
					db := m.databases[idx]
					m.inputMode = inputModeClone
					m.inputValue = db.name + "_clone"
					m.inputPrompt = "Clone " + db.name + " to: "
					m.confirmTarget = db.name
				}
			}
		case "x":
			switch m.focusedPane {
			case 2:
				if len(m.worktrees) > 0 {
					idx := m.selectedIndex[2]
					if idx < len(m.worktrees) {
						wt := m.worktrees[idx]
						if wt.isMain {
							m.showError = true
							m.errorMessage = "Cannot remove main worktree"
							return m, nil
						}
						m.confirmMode = confirmModeRemove
						m.confirmTarget = wt.branch
						m.confirmPath = wt.path
					}
				}
			case 3:
				if len(m.databases) > 0 {
					idx := m.selectedIndex[3]
					if idx < len(m.databases) {
						db := m.databases[idx]
						if db.isDefault {
							m.showError = true
							m.errorMessage = "Cannot drop the default database"
							return m, nil
						}
						m.confirmMode = confirmModeDrop
						m.confirmTarget = db.name
					}
				}
			}
		case "n":
			if m.focusedPane == 2 {
				m.inputMode = inputModeNewWT
				m.inputValue = ""
				m.inputPrompt = "New branch name: "
			}
		case "o":
			if m.focusedPane == 2 && len(m.worktrees) > 0 {
				idx := m.selectedIndex[2]
				if idx < len(m.worktrees) {
					wt := m.worktrees[idx]
					return m, m.openInTerminal(wt.path)
				}
			}
		case "i":
			if m.focusedPane == 4 && len(m.dumps) > 0 {
				idx := m.selectedIndex[4]
				if idx < len(m.dumps) {
					d := m.dumps[idx]
					m.inputMode = inputModeImport
					baseName := d.name
					if len(baseName) > 4 && baseName[len(baseName)-4:] == ".sql" {
						baseName = baseName[:len(baseName)-4]
					}
					m.inputValue = baseName
					m.inputPrompt = "Import " + d.name + " to database: "
					m.importDumpName = d.name
				}
			}
		}
	}
	return m, nil
}

type hideModalMsg struct{}

func (m Model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputMode = inputModeNone
		m.inputValue = ""
		return m, nil
	case "enter":
		if m.inputValue == "" {
			return m, nil
		}
		switch m.inputMode {
		case inputModeClone:
			m.inputMode = inputModeNone
			m.showModal = true
			m.modalTitle = "Cloning database..."
			m.modalMessage = m.confirmTarget + " → " + m.inputValue
			return m, m.cloneDatabase(m.confirmTarget, m.inputValue)
		case inputModeNewWT:
			m.inputMode = inputModeNone
			m.showModal = true
			m.modalTitle = "Creating worktree..."
			m.modalMessage = m.inputValue
			return m, m.createWorktree(m.inputValue)
		case inputModeImport:
			m.inputMode = inputModeNone
			m.showModal = true
			m.modalTitle = "Importing dump..."
			m.modalMessage = m.importDumpName + " → " + m.inputValue
			return m, m.importDump(m.inputValue, m.importDumpName)
		}
		m.inputMode = inputModeNone
	case "backspace":
		if len(m.inputValue) > 0 {
			m.inputValue = m.inputValue[:len(m.inputValue)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.inputValue += msg.String()
		}
	}
	return m, nil
}

func (m Model) handleConfirmMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		target := m.confirmTarget
		mode := m.confirmMode
		m.confirmMode = confirmModeNone
		m.confirmTarget = ""
		m.confirmPath = ""
		switch mode {
		case confirmModeDrop:
			m.showModal = true
			m.modalTitle = "Dropping database..."
			m.modalMessage = target
			return m, m.dropDatabase(target)
		case confirmModeRemove:
			m.showModal = true
			m.modalTitle = "Removing worktree..."
			m.modalMessage = target
			return m, m.removeWorktree(target)
		}
	case "n", "N", "esc":
		m.confirmMode = confirmModeNone
		m.confirmTarget = ""
		m.confirmPath = ""
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	paneWidth := m.width - 4

	infoContent := fmt.Sprintf("Project: %s | Type: %s | Compose: %s", m.projectName, m.projectType, m.projectStatus)
	infoPane := m.renderPane("Info", infoContent, 1, paneWidth)

	var wtItems []string
	for _, wt := range m.worktrees {
		prefix := "  "
		if wt.isMain {
			prefix = "* "
		}
		wtItems = append(wtItems, prefix+wt.branch)
	}
	worktreesPane := m.renderListPane("Worktrees", wtItems, 2, paneWidth)

	var dbItems []string
	for _, db := range m.databases {
		prefix := "  "
		if db.isDefault {
			prefix = "* "
		}
		dbItems = append(dbItems, prefix+db.name)
	}
	dbPane := m.renderListPane("Databases", dbItems, 3, paneWidth)

	dumpsPane := m.renderDumpsPane(m.dumps, 4, paneWidth)

	mainLayout := lipgloss.JoinVertical(lipgloss.Left, infoPane, worktreesPane, dbPane, dumpsPane)

	statusBar := statusBarStyle.Width(m.width).Render(m.statusBarText())

	baseView := lipgloss.JoinVertical(lipgloss.Top, mainLayout, statusBar)

	if m.showError {
		return m.renderErrorOverlay(baseView)
	}

	if m.showHelp {
		return m.renderHelpOverlay(baseView)
	}

	if m.showModal {
		return m.renderModalOverlay(baseView)
	}

	return baseView
}

func (m Model) renderPane(title, content string, paneNum, width int) string {
	style := paneStyle
	if m.focusedPane == paneNum {
		style = focusedPaneStyle
	}

	header := titleStyle.Render(fmt.Sprintf("[%d] %s", paneNum, title))
	body := lipgloss.NewStyle().Padding(0, 1).Render(content)

	pane := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return style.Width(width).Render(pane)
}

func (m Model) renderListPane(title string, items []string, paneNum, width int) string {
	style := paneStyle
	if m.focusedPane == paneNum {
		style = focusedPaneStyle
	}

	totalItems := len(items)
	scrollOffset := m.scrollOffset[paneNum]
	selectedIdx := m.selectedIndex[paneNum]

	scrollIndicator := ""
	if totalItems > maxVisibleLines {
		hasAbove := scrollOffset > 0
		hasBelow := scrollOffset+maxVisibleLines < totalItems
		if hasAbove && hasBelow {
			scrollIndicator = " ↑↓"
		} else if hasAbove {
			scrollIndicator = " ↑"
		} else if hasBelow {
			scrollIndicator = " ↓"
		}
	}

	header := titleStyle.Render(fmt.Sprintf("[%d] %s%s", paneNum, title, scrollIndicator))

	endIdx := scrollOffset + maxVisibleLines
	if endIdx > totalItems {
		endIdx = totalItems
	}

	var visibleLines []string
	for i := scrollOffset; i < endIdx; i++ {
		line := items[i]
		if i == selectedIdx && m.focusedPane == paneNum {
			line = selectedItemStyle.Render(line)
		}
		visibleLines = append(visibleLines, line)
	}

	content := strings.Join(visibleLines, "\n")
	if content == "" {
		content = "No items"
	}

	body := lipgloss.NewStyle().Padding(0, 1).Render(content)

	pane := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return style.Width(width).Render(pane)
}

func (m Model) clampScrollOffset(paneNum int) {
	selectedIdx := m.selectedIndex[paneNum]
	scrollOffset := m.scrollOffset[paneNum]

	if selectedIdx < scrollOffset {
		m.scrollOffset[paneNum] = selectedIdx
	} else if selectedIdx >= scrollOffset+maxVisibleLines {
		m.scrollOffset[paneNum] = selectedIdx - maxVisibleLines + 1
	}
}

func (m Model) renderDumpsPane(dumps []dumpInfo, paneNum, width int) string {
	style := paneStyle
	if m.focusedPane == paneNum {
		style = focusedPaneStyle
	}

	totalItems := len(dumps)
	scrollOffset := m.scrollOffset[paneNum]
	selectedIdx := m.selectedIndex[paneNum]

	scrollIndicator := ""
	if totalItems > maxVisibleLines {
		hasAbove := scrollOffset > 0
		hasBelow := scrollOffset+maxVisibleLines < totalItems
		if hasAbove && hasBelow {
			scrollIndicator = " ↑↓"
		} else if hasAbove {
			scrollIndicator = " ↑"
		} else if hasBelow {
			scrollIndicator = " ↓"
		}
	}

	header := titleStyle.Render(fmt.Sprintf("[%d] %s%s", paneNum, "Dumps", scrollIndicator))

	endIdx := scrollOffset + maxVisibleLines
	if endIdx > totalItems {
		endIdx = totalItems
	}

	contentWidth := width - 6

	var lines []string
	for i := scrollOffset; i < endIdx; i++ {
		d := dumps[i]

		nameWidth := contentWidth - 22
		dateWidth := 11
		sizeWidth := 8

		displayName := shortenFilename(d.name, nameWidth)

		nameStyle := lipgloss.NewStyle().Width(nameWidth)
		dateStyle := lipgloss.NewStyle().Width(dateWidth).Align(lipgloss.Right)
		sizeStyle := lipgloss.NewStyle().Width(sizeWidth).Align(lipgloss.Right)

		line := lipgloss.JoinHorizontal(lipgloss.Top,
			nameStyle.Render(displayName),
			dateStyle.Render(d.date),
			sizeStyle.Render(d.size),
		)

		if i == selectedIdx && m.focusedPane == paneNum {
			line = selectedItemStyle.Render(lipgloss.NewStyle().Width(contentWidth).Render(line))
		}
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	if content == "" {
		content = "No dumps"
	}

	body := lipgloss.NewStyle().Padding(0, 1).Render(content)

	pane := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return style.Width(width).Render(pane)
}

func (m Model) renderModalOverlay(baseView string) string {
	modalContent := fmt.Sprintf("%s\n\n%s", m.modalTitle, m.modalMessage)
	modal := modalStyle.Render(modalContent)

	overlay := lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)

	return baseView + "\n" + lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(lipgloss.NewStyle().Inherit(lipgloss.NewStyle()).Render(overlay))
}

func (m Model) renderErrorOverlay(baseView string) string {
	errorContent := fmt.Sprintf("✗ Error\n\n%s\n\nPress any key to dismiss", m.errorMessage)
	errorBox := modalStyle.
		BorderForeground(errorColor).
		Render(errorContent)

	overlay := lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		errorBox,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)

	return baseView + "\n" + lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(overlay)
}

func (m Model) renderHelpOverlay(baseView string) string {
	helpText := `Keyboard Shortcuts

Navigation:
  Tab/Shift+Tab  Cycle panes
  1-4           Jump to pane
  ↑/↓ or j/k    Navigate items

Databases (pane 3):
  d             Dump selected database
  c             Clone to new database
  x             Drop database (with confirmation)

Worktrees (pane 2):
  n             Create new worktree
  x             Remove worktree (with confirmation)
  o             Open in terminal

Dumps (pane 4):
  i             Import dump file
  x             Delete dump file

General:
  r             Refresh current pane
  R             Refresh all panes
  ?             Toggle this help
  q             Quit`

	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2).
		Width(50).
		Render(helpText)

	overlay := lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		helpBox,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)

	return baseView + "\n" + lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(overlay)
}

func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return err
	}
	return nil
}
