// Package github registers the GitHub toolset on the MCP server.
// API verified against mcp-go v0.52.0 and google/go-github v67.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	gogithub "github.com/google/go-github/v67/github"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
)

// Toolset wraps the GitHub client.
type Toolset struct {
	client *gogithub.Client
}

// New creates a GitHub Toolset authenticated with the given PAT token.
func New(token, host string) *Toolset {
	var httpClient *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		httpClient = oauth2.NewClient(context.Background(), ts)
	}
	client := gogithub.NewClient(httpClient)
	if host != "" && host != "https://api.github.com" {
		enterpriseURL := strings.TrimRight(host, "/") + "/api/v3/"
		client, _ = client.WithEnterpriseURLs(enterpriseURL, enterpriseURL)
	}
	return &Toolset{client: client}
}

// Register adds all GitHub tools to the MCP server.
func (t *Toolset) Register(srv *mcpserver.MCPServer) {
	t.registerSearchRepositories(srv)
	t.registerGetFileContents(srv)
	t.registerListIssues(srv)
	t.registerCreateIssue(srv)
	t.registerSearchCode(srv)
	t.registerListCommits(srv)
}

func (t *Toolset) registerSearchRepositories(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"github_search_repositories",
		mcp.WithDescription("Search GitHub repositories. Returns name, description, stars, language, URL. Side effect: NETWORK_READ."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query (e.g., 'fiber mcp language:go').")),
		mcp.WithNumber("limit", mcp.Description("Max results 1–30 (default 10).")),
	), t.handleSearchRepositories)
}

func (t *Toolset) handleSearchRepositories(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query is required"), nil
	}
	limit := req.GetInt("limit", 10)
	if limit < 1 || limit > 30 {
		limit = 10
	}
	opts := &gogithub.SearchOptions{ListOptions: gogithub.ListOptions{PerPage: limit}}
	result, _, err := t.client.Search.Repositories(ctx, query, opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("GitHub search failed: %s", err.Error())), nil
	}
	type item struct {
		FullName    string `json:"full_name"`
		Description string `json:"description"`
		Stars       int    `json:"stars"`
		Language    string `json:"language"`
		URL         string `json:"url"`
	}
	items := make([]item, 0, len(result.Repositories))
	for _, r := range result.Repositories {
		items = append(items, item{
			FullName:    r.GetFullName(),
			Description: r.GetDescription(),
			Stars:       r.GetStargazersCount(),
			Language:    r.GetLanguage(),
			URL:         r.GetHTMLURL(),
		})
	}
	out, _ := json.Marshal(map[string]any{"total": result.GetTotal(), "items": items})
	return mcp.NewToolResultText(string(out)), nil
}

func (t *Toolset) registerGetFileContents(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"github_get_file_contents",
		mcp.WithDescription("Read a file from a GitHub repository. Side effect: NETWORK_READ."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path (e.g., 'README.md').")),
		mcp.WithString("branch", mcp.Description("Branch or tag (default: default branch).")),
	), t.handleGetFileContents)
}

func (t *Toolset) handleGetFileContents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, _ := req.RequireString("owner")
	repo, _ := req.RequireString("repo")
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("owner, repo and path are required"), nil
	}
	branch := req.GetString("branch", "")
	fileContent, _, _, err := t.client.Repositories.GetContents(ctx, owner, repo, path,
		&gogithub.RepositoryContentGetOptions{Ref: branch})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot get file: %s", err.Error())), nil
	}
	content, err := fileContent.GetContent()
	if err != nil {
		return mcp.NewToolResultError("cannot decode file content"), nil
	}
	return mcp.NewToolResultText(content), nil
}

func (t *Toolset) registerListIssues(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"github_list_issues",
		mcp.WithDescription("List issues in a GitHub repository. Side effect: NETWORK_READ."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("state", mcp.Description("'open' (default), 'closed', or 'all'."), mcp.Enum("open", "closed", "all")),
		mcp.WithNumber("limit", mcp.Description("Max results 1–50 (default 20).")),
	), t.handleListIssues)
}

func (t *Toolset) handleListIssues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, _ := req.RequireString("owner")
	repo, err := req.RequireString("repo")
	if err != nil {
		return mcp.NewToolResultError("owner and repo are required"), nil
	}
	state := req.GetString("state", "open")
	limit := req.GetInt("limit", 20)
	if limit < 1 || limit > 50 {
		limit = 20
	}
	issues, _, err := t.client.Issues.ListByRepo(ctx, owner, repo, &gogithub.IssueListByRepoOptions{
		State: state, ListOptions: gogithub.ListOptions{PerPage: limit},
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot list issues: %s", err.Error())), nil
	}
	type item struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		URL    string `json:"url"`
		Author string `json:"author"`
	}
	items := make([]item, 0, len(issues))
	for _, i := range issues {
		if i.IsPullRequest() {
			continue
		}
		items = append(items, item{
			Number: i.GetNumber(), Title: i.GetTitle(),
			State: i.GetState(), URL: i.GetHTMLURL(),
			Author: i.GetUser().GetLogin(),
		})
	}
	out, _ := json.Marshal(items)
	return mcp.NewToolResultText(string(out)), nil
}

func (t *Toolset) registerCreateIssue(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"github_create_issue",
		mcp.WithDescription("Create a new issue in a GitHub repository. Side effect: NETWORK_WRITE."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Issue title.")),
		mcp.WithString("body", mcp.Description("Issue body (Markdown).")),
		mcp.WithString("labels", mcp.Description("Comma-separated labels.")),
	), t.handleCreateIssue)
}

func (t *Toolset) handleCreateIssue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, _ := req.RequireString("owner")
	repo, _ := req.RequireString("repo")
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("owner, repo and title are required"), nil
	}
	body := req.GetString("body", "")
	var labels []string
	if raw := req.GetString("labels", ""); raw != "" {
		for _, l := range strings.Split(raw, ",") {
			if t := strings.TrimSpace(l); t != "" {
				labels = append(labels, t)
			}
		}
	}
	issue, _, err := t.client.Issues.Create(ctx, owner, repo,
		&gogithub.IssueRequest{Title: &title, Body: &body, Labels: &labels})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot create issue: %s", err.Error())), nil
	}
	out, _ := json.Marshal(map[string]any{"number": issue.GetNumber(), "url": issue.GetHTMLURL()})
	return mcp.NewToolResultText(string(out)), nil
}

func (t *Toolset) registerSearchCode(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"github_search_code",
		mcp.WithDescription("Search code across GitHub repositories. Side effect: NETWORK_READ."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Code search query.")),
		mcp.WithNumber("limit", mcp.Description("Max results 1–20 (default 10).")),
	), t.handleSearchCode)
}

func (t *Toolset) handleSearchCode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query is required"), nil
	}
	limit := req.GetInt("limit", 10)
	if limit < 1 || limit > 20 {
		limit = 10
	}
	result, _, err := t.client.Search.Code(ctx, query,
		&gogithub.SearchOptions{ListOptions: gogithub.ListOptions{PerPage: limit}})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("code search failed: %s", err.Error())), nil
	}
	type item struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Repo string `json:"repo"`
		URL  string `json:"url"`
	}
	items := make([]item, 0, len(result.CodeResults))
	for _, c := range result.CodeResults {
		items = append(items, item{
			Name: c.GetName(), Path: c.GetPath(),
			Repo: c.GetRepository().GetFullName(), URL: c.GetHTMLURL(),
		})
	}
	out, _ := json.Marshal(map[string]any{"total": result.GetTotal(), "items": items})
	return mcp.NewToolResultText(string(out)), nil
}

func (t *Toolset) registerListCommits(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"github_list_commits",
		mcp.WithDescription("List recent commits for a branch. Side effect: NETWORK_READ."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner.")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name.")),
		mcp.WithString("branch", mcp.Description("Branch name (default: default branch).")),
		mcp.WithNumber("limit", mcp.Description("Max commits 1–50 (default 10).")),
	), t.handleListCommits)
}

func (t *Toolset) handleListCommits(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, _ := req.RequireString("owner")
	repo, err := req.RequireString("repo")
	if err != nil {
		return mcp.NewToolResultError("owner and repo are required"), nil
	}
	branch := req.GetString("branch", "")
	limit := req.GetInt("limit", 10)
	
	return t.HandleListCommits(ctx, owner, repo, branch, limit)
}

// HandleListCommits is the internal engine for fetching commit history.
// Exported so other specialized toolsets (like FiberExpert) can reuse it.
func (t *Toolset) HandleListCommits(ctx context.Context, owner, repo, branch string, limit int) (*mcp.CallToolResult, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}
	commits, _, err := t.client.Repositories.ListCommits(ctx, owner, repo,
		&gogithub.CommitsListOptions{SHA: branch, ListOptions: gogithub.ListOptions{PerPage: limit}})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot list commits: %s", err.Error())), nil
	}
	type item struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
		Author  string `json:"author"`
		URL     string `json:"url"`
	}
	items := make([]item, 0, len(commits))
	for _, c := range commits {
		msg := c.GetCommit().GetMessage()
		if len(msg) > 80 {
			msg = msg[:80] + "..."
		}
		sha := c.GetSHA()
		if len(sha) > 7 {
			sha = sha[:7]
		}
		items = append(items, item{
			SHA: sha, Message: msg,
			Author: c.GetCommit().GetAuthor().GetName(),
			URL:    c.GetHTMLURL(),
		})
	}
	out, _ := json.Marshal(items)
	return mcp.NewToolResultText(string(out)), nil
}
