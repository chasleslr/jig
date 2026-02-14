package git

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GHAvailable checks if the gh CLI is installed and authenticated
func GHAvailable() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}

// PR represents a GitHub pull request
type PR struct {
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	State       string   `json:"state"`
	URL         string   `json:"url"`
	HeadRefName string   `json:"headRefName"`
	BaseRefName string   `json:"baseRefName"`
	IsDraft     bool     `json:"isDraft"`
	Mergeable   string   `json:"mergeable"`
	Labels      []string `json:"labels"`
	Reviewers   []string `json:"reviewers"`
}

// PRComment represents a comment on a PR
type PRComment struct {
	ID        int    `json:"id"`
	Body      string `json:"body"`
	Author    string `json:"author"`
	Path      string `json:"path"`
	Line      int    `json:"line"`
	State     string `json:"state"` // PENDING, COMMENTED, APPROVED, etc.
	CreatedAt string `json:"createdAt"`
}

// CreatePR creates a new pull request
func CreatePR(title, body, baseBranch string, draft bool) (*PR, error) {
	args := []string{"pr", "create", "--title", title, "--body", body}

	if baseBranch != "" {
		args = append(args, "--base", baseBranch)
	}

	if draft {
		args = append(args, "--draft")
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %s: %w", string(output), err)
	}

	// Parse the URL from output to get PR number
	url := strings.TrimSpace(string(output))

	// Get full PR info
	return GetPRByURL(url)
}

// GetPR returns the PR for the current branch
func GetPR() (*PR, error) {
	cmd := exec.Command("gh", "pr", "view", "--json",
		"number,title,body,state,url,headRefName,baseRefName,isDraft,mergeable")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	var pr PR
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR: %w", err)
	}

	return &pr, nil
}

// GetPRByNumber returns a PR by number
func GetPRByNumber(number int) (*PR, error) {
	cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%d", number), "--json",
		"number,title,body,state,url,headRefName,baseRefName,isDraft,mergeable")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	var pr PR
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR: %w", err)
	}

	return &pr, nil
}

// GetPRByURL returns a PR by URL
func GetPRByURL(url string) (*PR, error) {
	cmd := exec.Command("gh", "pr", "view", url, "--json",
		"number,title,body,state,url,headRefName,baseRefName,isDraft,mergeable")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	var pr PR
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR: %w", err)
	}

	return &pr, nil
}

// GetPRForBranch returns the PR for a specific branch
func GetPRForBranch(branch string) (*PR, error) {
	cmd := exec.Command("gh", "pr", "view", branch, "--json",
		"number,title,body,state,url,headRefName,baseRefName,isDraft,mergeable")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil // No PR for this branch
	}

	var pr PR
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR: %w", err)
	}

	return &pr, nil
}

// GetPRComments returns unresolved review comments on a PR
func GetPRComments(prNumber int) ([]PRComment, error) {
	// Use gh api to get review comments
	cmd := exec.Command("gh", "api",
		fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/comments", prNumber),
		"--jq", `.[] | {id: .id, body: .body, author: .user.login, path: .path, line: .line, createdAt: .created_at}`)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR comments: %w", err)
	}

	var comments []PRComment
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}
		var comment PRComment
		if err := json.Unmarshal([]byte(line), &comment); err != nil {
			continue
		}
		comments = append(comments, comment)
	}

	return comments, nil
}

// GetPRReviewThreads returns review threads (including resolved status)
func GetPRReviewThreads(prNumber int) ([]PRComment, error) {
	// Use GraphQL to get review threads with resolved status
	query := `
	query($owner: String!, $repo: String!, $number: Int!) {
		repository(owner: $owner, name: $repo) {
			pullRequest(number: $number) {
				reviewThreads(first: 100) {
					nodes {
						isResolved
						comments(first: 10) {
							nodes {
								id
								body
								author { login }
								path
								line
								createdAt
							}
						}
					}
				}
			}
		}
	}`

	// Get owner and repo
	cmd := exec.Command("gh", "repo", "view", "--json", "owner,name")
	repoOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo info: %w", err)
	}

	var repoInfo struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(repoOutput, &repoInfo); err != nil {
		return nil, err
	}

	cmd = exec.Command("gh", "api", "graphql",
		"-F", fmt.Sprintf("owner=%s", repoInfo.Owner.Login),
		"-F", fmt.Sprintf("repo=%s", repoInfo.Name),
		"-F", fmt.Sprintf("number=%d", prNumber),
		"-f", fmt.Sprintf("query=%s", query))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR review threads: %w", err)
	}

	var result struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						Nodes []struct {
							IsResolved bool `json:"isResolved"`
							Comments   struct {
								Nodes []struct {
									ID     string `json:"id"`
									Body   string `json:"body"`
									Author struct {
										Login string `json:"login"`
									} `json:"author"`
									Path      string `json:"path"`
									Line      int    `json:"line"`
									CreatedAt string `json:"createdAt"`
								} `json:"nodes"`
							} `json:"comments"`
						} `json:"nodes"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse review threads: %w", err)
	}

	var comments []PRComment
	for _, thread := range result.Data.Repository.PullRequest.ReviewThreads.Nodes {
		if thread.IsResolved {
			continue // Skip resolved threads
		}
		for _, c := range thread.Comments.Nodes {
			comments = append(comments, PRComment{
				Body:      c.Body,
				Author:    c.Author.Login,
				Path:      c.Path,
				Line:      c.Line,
				CreatedAt: c.CreatedAt,
			})
		}
	}

	return comments, nil
}

// MergePR merges a pull request
func MergePR(prNumber int, method string, deleteAfter bool) error {
	args := []string{"pr", "merge", fmt.Sprintf("%d", prNumber)}

	switch method {
	case "merge":
		args = append(args, "--merge")
	case "squash":
		args = append(args, "--squash")
	case "rebase":
		args = append(args, "--rebase")
	default:
		args = append(args, "--merge")
	}

	if deleteAfter {
		args = append(args, "--delete-branch")
	}

	cmd := exec.Command("gh", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to merge PR: %s: %w", string(output), err)
	}

	return nil
}

// ClosePR closes a pull request without merging
func ClosePR(prNumber int) error {
	cmd := exec.Command("gh", "pr", "close", fmt.Sprintf("%d", prNumber))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to close PR: %s: %w", string(output), err)
	}
	return nil
}

// UpdatePR updates PR title and body
func UpdatePR(prNumber int, title, body string) error {
	args := []string{"pr", "edit", fmt.Sprintf("%d", prNumber)}

	if title != "" {
		args = append(args, "--title", title)
	}
	if body != "" {
		args = append(args, "--body", body)
	}

	cmd := exec.Command("gh", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update PR: %s: %w", string(output), err)
	}
	return nil
}

// SetPRReady marks a draft PR as ready for review
func SetPRReady(prNumber int) error {
	cmd := exec.Command("gh", "pr", "ready", fmt.Sprintf("%d", prNumber))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mark PR ready: %s: %w", string(output), err)
	}
	return nil
}

// GetRepoInfo returns the owner and repo name
func GetRepoInfo() (owner, repo string, err error) {
	cmd := exec.Command("gh", "repo", "view", "--json", "owner,name")
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get repo info: %w", err)
	}

	var info struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(output, &info); err != nil {
		return "", "", err
	}

	return info.Owner.Login, info.Name, nil
}

// OpenPRInBrowser opens the PR for a branch in the default browser
func OpenPRInBrowser(branch string) error {
	cmd := exec.Command("gh", "pr", "view", branch, "--web")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to open PR: %s: %w", string(output), err)
	}
	return nil
}

// GetCIStatus returns the CI status for the current branch
func GetCIStatus() (string, error) {
	cmd := exec.Command("gh", "pr", "checks", "--json", "state")
	output, err := cmd.Output()
	if err != nil {
		return "unknown", nil
	}

	var checks []struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(output, &checks); err != nil {
		return "unknown", nil
	}

	// Return worst status
	for _, check := range checks {
		if check.State == "FAILURE" {
			return "failure", nil
		}
	}
	for _, check := range checks {
		if check.State == "PENDING" {
			return "pending", nil
		}
	}

	return "success", nil
}
