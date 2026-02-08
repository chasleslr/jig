package git

import (
	"testing"
)

func TestRealClientImplementsInterface(t *testing.T) {
	// Verify RealClient implements Client interface at compile time
	var _ Client = (*RealClient)(nil)

	// Verify we can create and use a RealClient
	client := &RealClient{}
	// The client should be usable (we can't test actual gh CLI calls in unit tests)
	_ = client
}

func TestDefaultClientIsSet(t *testing.T) {
	if DefaultClient == nil {
		t.Error("DefaultClient should be set")
	}

	// Verify DefaultClient is a RealClient
	_, ok := DefaultClient.(*RealClient)
	if !ok {
		t.Error("DefaultClient should be a *RealClient")
	}
}

func TestDefaultClientCanBeReplaced(t *testing.T) {
	// Save original
	original := DefaultClient

	// Replace with a mock-like implementation
	mock := &testMockClient{available: true}
	DefaultClient = mock

	// Verify replacement
	if DefaultClient != mock {
		t.Error("DefaultClient should be replaceable")
	}

	// Verify we can call methods on the replaced client
	if !DefaultClient.Available() {
		t.Error("replaced client should return expected availability")
	}

	// Restore original
	DefaultClient = original
}

// testMockClient is a minimal mock for testing DefaultClient replacement
type testMockClient struct {
	available bool
}

func (m *testMockClient) Available() bool {
	return m.available
}

func (m *testMockClient) GetCurrentBranch() (string, error) {
	return "test-branch", nil
}

func (m *testMockClient) GetPR() (*PR, error) {
	return nil, nil
}

func (m *testMockClient) GetPRByNumber(number int) (*PR, error) {
	return nil, nil
}

func (m *testMockClient) GetPRForBranch(branch string) (*PR, error) {
	return nil, nil
}

func (m *testMockClient) CreatePR(title, body, baseBranch string, draft bool) (*PR, error) {
	return nil, nil
}

func (m *testMockClient) MergePR(number int, method string, deleteAfter bool) error {
	return nil
}

func (m *testMockClient) GetPRComments(prNumber int) ([]PRComment, error) {
	return nil, nil
}

func (m *testMockClient) GetPRReviewThreads(prNumber int) ([]PRComment, error) {
	return nil, nil
}

func (m *testMockClient) GetCIStatus() (string, error) {
	return "success", nil
}
