package git

// Client defines the interface for GitHub operations via gh CLI.
// This interface enables dependency injection and testing without
// requiring the actual gh CLI to be installed.
type Client interface {
	// Available checks if the gh CLI is installed and authenticated
	Available() bool

	// GetCurrentBranch returns the name of the current git branch
	GetCurrentBranch() (string, error)

	// PR operations
	GetPR() (*PR, error)
	GetPRByNumber(number int) (*PR, error)
	GetPRForBranch(branch string) (*PR, error)
	CreatePR(title, body, baseBranch string, draft bool) (*PR, error)
	MergePR(number int, method string, deleteAfter bool) error

	// PR review operations
	GetPRComments(prNumber int) ([]PRComment, error)
	GetPRReviewThreads(prNumber int) ([]PRComment, error)

	// CI operations
	GetCIStatus() (string, error)
}

// RealClient implements Client using the actual gh CLI
type RealClient struct{}

// Ensure RealClient implements Client
var _ Client = (*RealClient)(nil)

func (c *RealClient) Available() bool {
	return GHAvailable()
}

func (c *RealClient) GetCurrentBranch() (string, error) {
	return GetCurrentBranch()
}

func (c *RealClient) GetPR() (*PR, error) {
	return GetPR()
}

func (c *RealClient) GetPRByNumber(number int) (*PR, error) {
	return GetPRByNumber(number)
}

func (c *RealClient) GetPRForBranch(branch string) (*PR, error) {
	return GetPRForBranch(branch)
}

func (c *RealClient) CreatePR(title, body, baseBranch string, draft bool) (*PR, error) {
	return CreatePR(title, body, baseBranch, draft)
}

func (c *RealClient) MergePR(number int, method string, deleteAfter bool) error {
	return MergePR(number, method, deleteAfter)
}

func (c *RealClient) GetPRComments(prNumber int) ([]PRComment, error) {
	return GetPRComments(prNumber)
}

func (c *RealClient) GetPRReviewThreads(prNumber int) ([]PRComment, error) {
	return GetPRReviewThreads(prNumber)
}

func (c *RealClient) GetCIStatus() (string, error) {
	return GetCIStatus()
}

// DefaultClient is the default Client implementation used by the application.
// It can be replaced in tests with a mock implementation.
var DefaultClient Client = &RealClient{}
