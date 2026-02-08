// Package mock provides a mock implementation of git.Client for testing.
package mock

import (
	"fmt"
	"sync"

	"github.com/charleslr/jig/internal/git"
)

// Client is a mock implementation of git.Client for testing.
// It allows tests to configure responses and verify calls.
type Client struct {
	mu sync.RWMutex

	// Configuration
	IsAvailable   bool
	CurrentBranch string

	// PR storage: branch name -> PR
	PRsByBranch map[string]*git.PR
	// PR storage: number -> PR
	PRsByNumber map[int]*git.PR

	// Comments storage: PR number -> comments
	Comments      map[int][]git.PRComment
	ReviewThreads map[int][]git.PRComment

	// CI status
	CIStatus string

	// Error injection for testing error paths
	AvailableError        error
	GetCurrentBranchError error
	GetPRError            error
	GetPRByNumberError    error
	GetPRForBranchError   error
	CreatePRError         error
	MergePRError          error
	GetPRCommentsError    error
	GetPRReviewThreadsError error
	GetCIStatusError      error

	// Call tracking
	CreatePRCalls []CreatePRCall
	MergePRCalls  []MergePRCall
}

// CreatePRCall records a call to CreatePR
type CreatePRCall struct {
	Title      string
	Body       string
	BaseBranch string
	Draft      bool
}

// MergePRCall records a call to MergePR
type MergePRCall struct {
	Number      int
	Method      string
	DeleteAfter bool
}

// Ensure Client implements git.Client
var _ git.Client = (*Client)(nil)

// NewClient creates a new mock client with sensible defaults
func NewClient() *Client {
	return &Client{
		IsAvailable:   true,
		CurrentBranch: "main",
		PRsByBranch:   make(map[string]*git.PR),
		PRsByNumber:   make(map[int]*git.PR),
		Comments:      make(map[int][]git.PRComment),
		ReviewThreads: make(map[int][]git.PRComment),
		CIStatus:      "success",
	}
}

// Available returns the configured availability
func (c *Client) Available() bool {
	if c.AvailableError != nil {
		return false
	}
	return c.IsAvailable
}

// GetCurrentBranch returns the configured current branch
func (c *Client) GetCurrentBranch() (string, error) {
	if c.GetCurrentBranchError != nil {
		return "", c.GetCurrentBranchError
	}
	return c.CurrentBranch, nil
}

// GetPR returns the PR for the current branch
func (c *Client) GetPR() (*git.PR, error) {
	if c.GetPRError != nil {
		return nil, c.GetPRError
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	pr, ok := c.PRsByBranch[c.CurrentBranch]
	if !ok {
		return nil, fmt.Errorf("no PR found for current branch")
	}
	return pr, nil
}

// GetPRByNumber returns a PR by its number
func (c *Client) GetPRByNumber(number int) (*git.PR, error) {
	if c.GetPRByNumberError != nil {
		return nil, c.GetPRByNumberError
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	pr, ok := c.PRsByNumber[number]
	if !ok {
		return nil, fmt.Errorf("PR #%d not found", number)
	}
	return pr, nil
}

// GetPRForBranch returns the PR for a specific branch
func (c *Client) GetPRForBranch(branch string) (*git.PR, error) {
	if c.GetPRForBranchError != nil {
		return nil, c.GetPRForBranchError
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	pr, ok := c.PRsByBranch[branch]
	if !ok {
		return nil, nil // No PR for this branch (not an error)
	}
	return pr, nil
}

// CreatePR creates a mock PR and stores it
func (c *Client) CreatePR(title, body, baseBranch string, draft bool) (*git.PR, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Record the call
	c.CreatePRCalls = append(c.CreatePRCalls, CreatePRCall{
		Title:      title,
		Body:       body,
		BaseBranch: baseBranch,
		Draft:      draft,
	})

	if c.CreatePRError != nil {
		return nil, c.CreatePRError
	}

	// Generate a PR number
	prNumber := len(c.PRsByNumber) + 1

	pr := &git.PR{
		Number:      prNumber,
		Title:       title,
		Body:        body,
		State:       "OPEN",
		URL:         fmt.Sprintf("https://github.com/test/repo/pull/%d", prNumber),
		HeadRefName: c.CurrentBranch,
		BaseRefName: baseBranch,
		IsDraft:     draft,
		Mergeable:   "MERGEABLE",
	}

	c.PRsByBranch[c.CurrentBranch] = pr
	c.PRsByNumber[prNumber] = pr

	return pr, nil
}

// MergePR records the merge call
func (c *Client) MergePR(number int, method string, deleteAfter bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.MergePRCalls = append(c.MergePRCalls, MergePRCall{
		Number:      number,
		Method:      method,
		DeleteAfter: deleteAfter,
	})

	if c.MergePRError != nil {
		return c.MergePRError
	}

	// Update PR state
	if pr, ok := c.PRsByNumber[number]; ok {
		pr.State = "MERGED"
	}

	return nil
}

// GetPRComments returns mock comments for a PR
func (c *Client) GetPRComments(prNumber int) ([]git.PRComment, error) {
	if c.GetPRCommentsError != nil {
		return nil, c.GetPRCommentsError
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.Comments[prNumber], nil
}

// GetPRReviewThreads returns mock review threads for a PR
func (c *Client) GetPRReviewThreads(prNumber int) ([]git.PRComment, error) {
	if c.GetPRReviewThreadsError != nil {
		return nil, c.GetPRReviewThreadsError
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.ReviewThreads[prNumber], nil
}

// GetCIStatus returns the configured CI status
func (c *Client) GetCIStatus() (string, error) {
	if c.GetCIStatusError != nil {
		return "", c.GetCIStatusError
	}
	return c.CIStatus, nil
}

// Helper methods for test setup

// AddPR adds a PR to the mock
func (c *Client) AddPR(pr *git.PR) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.PRsByNumber[pr.Number] = pr
	if pr.HeadRefName != "" {
		c.PRsByBranch[pr.HeadRefName] = pr
	}
}

// AddComment adds a comment to a PR
func (c *Client) AddComment(prNumber int, comment git.PRComment) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Comments[prNumber] = append(c.Comments[prNumber], comment)
}

// AddReviewThread adds a review thread to a PR
func (c *Client) AddReviewThread(prNumber int, comment git.PRComment) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ReviewThreads[prNumber] = append(c.ReviewThreads[prNumber], comment)
}

// Reset clears all state for reuse in tests
func (c *Client) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.PRsByBranch = make(map[string]*git.PR)
	c.PRsByNumber = make(map[int]*git.PR)
	c.Comments = make(map[int][]git.PRComment)
	c.ReviewThreads = make(map[int][]git.PRComment)
	c.CreatePRCalls = nil
	c.MergePRCalls = nil

	// Reset errors
	c.AvailableError = nil
	c.GetCurrentBranchError = nil
	c.GetPRError = nil
	c.GetPRByNumberError = nil
	c.GetPRForBranchError = nil
	c.CreatePRError = nil
	c.MergePRError = nil
	c.GetPRCommentsError = nil
	c.GetPRReviewThreadsError = nil
	c.GetCIStatusError = nil
}
