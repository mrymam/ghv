package command

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mrymam/ghv/internal/format"
	gh "github.com/mrymam/ghv/pkg/github"
	"github.com/spf13/cobra"
)

const defaultPollInterval = 5 * time.Minute

// section represents a tab in the TUI.
type section struct {
	title     string
	prs       []gh.PR
	statusMap map[string]gh.ReviewStatus
}

type tuiModel struct {
	sections     []section
	activeTab    int
	cursor       int
	width        int
	height       int
	loading      bool
	username     string
	org          string
	pollInterval time.Duration
}

// Messages
type dataLoadedMsg struct {
	sections []section
}

type tickMsg struct{}

func initialModel(username, org string, pollInterval time.Duration) tuiModel {
	return tuiModel{
		loading:      true,
		username:     username,
		org:          org,
		pollInterval: pollInterval,
	}
}

func loadData(username, orgQ, org string) tea.Cmd {
	return func() tea.Msg {
		ignore := gh.ParseIgnoredReviewers(os.Getenv("GHV_IGNORE_REVIEWERS"))
		var myPRs, reviewPRs []gh.PR
		var myStatus, revStatus map[string]gh.ReviewStatus
		var wg sync.WaitGroup

		// Fetch My PRs
		wg.Add(1)
		go func() {
			defer wg.Done()
			var authorRes, assigneeRes gh.FetchResult
			var wg2 sync.WaitGroup
			wg2.Add(2)
			go func() {
				defer wg2.Done()
				prs, err := gh.SearchPRs(fmt.Sprintf("author:%s%s", username, orgQ))
				authorRes = gh.FetchResult{PRs: prs, Err: err}
			}()
			go func() {
				defer wg2.Done()
				prs, err := gh.SearchPRs(fmt.Sprintf("assignee:%s%s", username, orgQ))
				assigneeRes = gh.FetchResult{PRs: prs, Err: err}
			}()
			wg2.Wait()
			myPRs = gh.MergePRs(authorRes.PRs, assigneeRes.PRs)
			gh.SortPRsByUpdated(myPRs)
			myStatus = gh.FetchAllReviewStatuses(myPRs, ignore)
		}()

		// Fetch Review PRs
		wg.Add(1)
		go func() {
			defer wg.Done()
			prs, _ := gh.SearchPRs(fmt.Sprintf("review-requested:%s%s", username, orgQ))
			gh.SortPRsByUpdated(prs)
			reviewPRs = prs
			revStatus = gh.FetchAllReviewStatuses(prs, ignore)
		}()

		wg.Wait()

		sections := []section{
			{title: "🔧 自分のPR", prs: myPRs, statusMap: myStatus},
			{title: "👀 レビュー待ち", prs: reviewPRs, statusMap: revStatus},
		}
		return dataLoadedMsg{sections: sections}
	}
}

func (m tuiModel) scheduleTick() tea.Cmd {
	if m.pollInterval <= 0 {
		return nil
	}
	return tea.Tick(m.pollInterval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m tuiModel) Init() tea.Cmd {
	_, orgQ := ResolveOrg()
	org, _ := ResolveOrg()
	return tea.Batch(loadData(m.username, orgQ, org), m.scheduleTick())
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.loading = true
		_, orgQ := ResolveOrg()
		org, _ := ResolveOrg()
		return m, loadData(m.username, orgQ, org)

	case dataLoadedMsg:
		m.sections = msg.sections
		m.loading = false
		// Keep cursor in bounds after reload
		if len(m.sections) > 0 {
			if m.activeTab >= len(m.sections) {
				m.activeTab = 0
				m.cursor = 0
			} else if m.cursor >= len(m.sections[m.activeTab].prs) {
				m.cursor = len(m.sections[m.activeTab].prs) - 1
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		}
		return m, m.scheduleTick()

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "l", "right":
			m.activeTab = (m.activeTab + 1) % len(m.sections)
			m.cursor = 0
		case "shift+tab", "h", "left":
			m.activeTab = (m.activeTab - 1 + len(m.sections)) % len(m.sections)
			m.cursor = 0
		case "j", "down":
			sec := m.sections[m.activeTab]
			if m.cursor < len(sec.prs)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			sec := m.sections[m.activeTab]
			if m.cursor < len(sec.prs) {
				openBrowser(sec.prs[m.cursor].HTMLURL)
			}
		case "r":
			m.loading = true
			_, orgQ := ResolveOrg()
			org, _ := ResolveOrg()
			return m, loadData(m.username, orgQ, org)
		}
	}
	return m, nil
}

func (m tuiModel) View() string {
	if m.loading && len(m.sections) == 0 {
		return "\n  Loading...\n"
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	header := headerStyle.Render(fmt.Sprintf("📋 GitHub Dashboard for @%s", m.username))
	if m.org != "" {
		header += lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf(" (org: %s)", m.org))
	}
	if m.loading && len(m.sections) > 0 {
		header += lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("  リロード中...")
	}
	b.WriteString(header + "\n\n")

	// Tabs
	activeTabStyle := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("4")).Foreground(lipgloss.Color("15")).Padding(0, 1)
	inactiveTabStyle := lipgloss.NewStyle().Faint(true).Padding(0, 1)
	var tabs []string
	for i, sec := range m.sections {
		label := fmt.Sprintf("%s (%d)", sec.title, len(sec.prs))
		if i == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(label))
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...) + "\n\n")

	// PR list
	sec := m.sections[m.activeTab]
	if len(sec.prs) == 0 {
		b.WriteString("  なし\n")
	} else {
		selectedStyle := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("8"))
		normalStyle := lipgloss.NewStyle()
		statusApproved := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
		statusChanges := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		statusReviewed := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
		statusOpen := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
		statusDraft := lipgloss.NewStyle().Faint(true)
		authorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
		dimStyle := lipgloss.NewStyle().Faint(true)

		maxVisible := m.height - 8
		if maxVisible < 5 {
			maxVisible = 5
		}

		start := 0
		if m.cursor >= maxVisible {
			start = m.cursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(sec.prs) {
			end = len(sec.prs)
		}

		for i := start; i < end; i++ {
			pr := sec.prs[i]
			repo := format.Truncate(pr.RepoName(), 20)
			title := format.Truncate(pr.Title, 50)
			age := format.FormatAge(pr.UpdatedAt)

			var line string
			if sec.statusMap != nil {
				status, _ := gh.CombinedStatus(pr, sec.statusMap[pr.HTMLURL])
				var styledStatus string
				switch {
				case strings.HasPrefix(status, "approved"):
					styledStatus = statusApproved.Render(fmt.Sprintf("%-25s", status))
				case strings.HasPrefix(status, "changes"):
					styledStatus = statusChanges.Render(fmt.Sprintf("%-25s", status))
				case strings.HasPrefix(status, "reviewed"):
					styledStatus = statusReviewed.Render(fmt.Sprintf("%-25s", status))
				case status == "draft":
					styledStatus = statusDraft.Render(fmt.Sprintf("%-25s", status))
				default:
					styledStatus = statusOpen.Render(fmt.Sprintf("%-25s", status))
				}
				line = fmt.Sprintf("  %s  %-20s  %s  %s", styledStatus, repo, dimStyle.Render(fmt.Sprintf("%-8s", age)), title)
			} else {
				line = fmt.Sprintf("  %-20s  %s  %s  %s", repo, authorStyle.Render(fmt.Sprintf("%-15s", pr.User.Login)), dimStyle.Render(fmt.Sprintf("%-8s", age)), title)
			}

			if i == m.cursor {
				b.WriteString(selectedStyle.Render("▸"+line) + "\n")
			} else {
				b.WriteString(normalStyle.Render(" "+line) + "\n")
			}
		}

		if len(sec.prs) > maxVisible {
			b.WriteString(dimStyle.Render(fmt.Sprintf("\n  %d/%d", m.cursor+1, len(sec.prs))) + "\n")
		}
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Faint(true)
	b.WriteString(helpStyle.Render("  ↑↓/jk: 移動  ←→/hl/tab: タブ切替  enter: ブラウザで開く  r: リロード  q: 終了"))
	b.WriteString("\n")

	return b.String()
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

// NewTUICmd creates the tui subcommand.
func NewTUICmd() *cobra.Command {
	var pollFlag time.Duration
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "インタラクティブなTUI表示",
		RunE: func(cmd *cobra.Command, args []string) error {
			username, err := gh.GetUsername()
			if err != nil {
				return err
			}
			org, _ := ResolveOrg()
			var interval time.Duration
			if cmd.Flags().Changed("poll") {
				interval = pollFlag
			}
			m := initialModel(username, org, interval)
			p := tea.NewProgram(m, tea.WithAltScreen())
			_, err = p.Run()
			return err
		},
	}
	cmd.Flags().DurationVar(&pollFlag, "poll", defaultPollInterval, "自動リロード間隔 (例: 5m, 30s). 指定時のみ有効")
	return cmd
}
