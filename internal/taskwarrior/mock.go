package taskwarrior

import (
	"context"
	"fmt"
)

type MockClient struct {
	AddFn      func(ctx context.Context, req AddRequest) (string, error)
	ExportFn   func(ctx context.Context, f Filter) ([]Task, error)
	TagsFn     func(ctx context.Context) ([]string, error)
	ProjectsFn func(ctx context.Context) ([]string, error)
	CompleteFn func(ctx context.Context, uuid string) error
	ModifyFn   func(ctx context.Context, uuid string, req AddRequest) error
	VersionFn  func(ctx context.Context) (string, error)
}

var _ Client = (*MockClient)(nil)

func (m *MockClient) Add(ctx context.Context, req AddRequest) (string, error) {
	if m.AddFn != nil {
		return m.AddFn(ctx, req)
	}
	return "", fmt.Errorf("MockClient.Add not implemented")
}

func (m *MockClient) Export(ctx context.Context, f Filter) ([]Task, error) {
	if m.ExportFn != nil {
		return m.ExportFn(ctx, f)
	}
	return nil, fmt.Errorf("MockClient.Export not implemented")
}

func (m *MockClient) Tags(ctx context.Context) ([]string, error) {
	if m.TagsFn != nil {
		return m.TagsFn(ctx)
	}
	return nil, nil // sensible default: empty list
}

func (m *MockClient) Projects(ctx context.Context) ([]string, error) {
	if m.ProjectsFn != nil {
		return m.ProjectsFn(ctx)
	}
	return nil, nil // sensible default: empty list
}

func (m *MockClient) Complete(ctx context.Context, uuid string) error {
	if m.CompleteFn != nil {
		return m.CompleteFn(ctx, uuid)
	}
	return fmt.Errorf("MockClient.Complete not implemented")
}

func (m *MockClient) Modify(ctx context.Context, uuid string, req AddRequest) error {
	if m.ModifyFn != nil {
		return m.ModifyFn(ctx, uuid, req)
	}
	return fmt.Errorf("MockClient.Modify not implemented")
}

func (m *MockClient) Version(ctx context.Context) (string, error) {
	if m.VersionFn != nil {
		return m.VersionFn(ctx)
	}
	return "mock-0.0.0", nil
}
