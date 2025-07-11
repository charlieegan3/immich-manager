// Package applier provides functionality to apply plans to the Immich API.
package applier

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"immich-manager/pkg/immich"
	"immich-manager/pkg/plan"
)

// Applier applies plans to the Immich API.
type Applier struct {
	client *immich.Client
}

// ApplyOptions contains options for the Apply operation.
type ApplyOptions struct {
	DryRun bool
	Writer io.Writer // Used for dry run output
}

// DefaultApplyOptions returns the default options for Apply.
func DefaultApplyOptions() *ApplyOptions {
	return &ApplyOptions{
		DryRun: false,
		Writer: nil,
	}
}

// NewApplier creates a new plan applier.
func NewApplier(client *immich.Client) *Applier {
	return &Applier{
		client: client,
	}
}

// Apply executes all operations in the plan.
func (a *Applier) Apply(p *plan.Plan, opts *ApplyOptions) error {
	if opts == nil {
		opts = DefaultApplyOptions()
	}

	if opts.DryRun {
		return a.dryRunApply(p, opts.Writer)
	}

	for i, op := range p.Operations {
		for j, req := range op.Apply {
			request, err := a.client.NewRequest(req.Method, req.Path, req.Body)
			if err != nil {
				return fmt.Errorf("creating request %d for operation %d: %w", j, i, err)
			}

			if err := a.client.Do(request, nil); err != nil {
				return fmt.Errorf("executing request %d for operation %d: %w", j, i, err)
			}
		}
	}

	return nil
}

// Revert executes all revert operations in the plan in reverse order.
func (a *Applier) Revert(p *plan.Plan, opts *ApplyOptions) error {
	if opts == nil {
		opts = DefaultApplyOptions()
	}

	if opts.DryRun {
		return a.dryRunRevert(p, opts.Writer)
	}

	// Execute operations in reverse order
	for i := len(p.Operations) - 1; i >= 0; i-- {
		op := p.Operations[i]

		// Execute each revert request in order
		for j, req := range op.Revert {
			request, err := a.client.NewRequest(req.Method, req.Path, req.Body)
			if err != nil {
				return fmt.Errorf("creating revert request %d for operation %d: %w", j, i, err)
			}

			if err := a.client.Do(request, nil); err != nil {
				return fmt.Errorf("executing revert request %d for operation %d: %w", j, i, err)
			}
		}
	}

	return nil
}

// dryRunApply simulates applying the plan without making actual API calls.
func (*Applier) dryRunApply(p *plan.Plan, w io.Writer) error {
	if w == nil {
		return errors.New("writer is required for dry run")
	}

	// Count total requests
	totalRequests := 0
	for _, op := range p.Operations {
		totalRequests += len(op.Apply)
	}

	if _, err := fmt.Fprintf(w, "Dry run mode: would execute %d operations with %d total requests\n",
		len(p.Operations), totalRequests); err != nil {
		return fmt.Errorf("writing dry run summary: %w", err)
	}

	for i, op := range p.Operations {
		if _, err := fmt.Fprintf(w, "Operation %d: %d requests\n", i+1, len(op.Apply)); err != nil {
			return fmt.Errorf("writing operation summary: %w", err)
		}

		for j, req := range op.Apply {
			if _, err := fmt.Fprintf(w, "  Request %d.%d: %s %s\n", i+1, j+1, req.Method, req.Path); err != nil {
				return fmt.Errorf("writing request summary: %w", err)
			}

			if req.Body != nil {
				bodyJSON, err := json.MarshalIndent(req.Body, "    ", "  ")
				if err != nil {
					return fmt.Errorf("marshaling body for operation %d request %d: %w", i, j, err)
				}

				if _, err := fmt.Fprintf(w, "    Body: %s\n", bodyJSON); err != nil {
					return fmt.Errorf("writing request body: %w", err)
				}
			}
		}
	}

	return nil
}

// dryRunRevert simulates reverting the plan without making actual API calls.
func (*Applier) dryRunRevert(p *plan.Plan, w io.Writer) error {
	if w == nil {
		return errors.New("writer is required for dry run")
	}

	// Count total requests
	totalRequests := 0
	for _, op := range p.Operations {
		totalRequests += len(op.Revert)
	}

	if _, err := fmt.Fprintf(w, "Dry run mode: would revert %d operations with %d total requests\n",
		len(p.Operations), totalRequests); err != nil {
		return fmt.Errorf("writing dry run revert summary: %w", err)
	}

	// Operations are processed in reverse order for revert
	for i := len(p.Operations) - 1; i >= 0; i-- {
		op := p.Operations[i]
		opNumber := len(p.Operations) - i

		if _, err := fmt.Fprintf(w, "Operation %d: %d requests\n", opNumber, len(op.Revert)); err != nil {
			return fmt.Errorf("writing revert operation summary: %w", err)
		}

		// Requests within an operation are processed in original order
		for j, req := range op.Revert {
			if _, err := fmt.Fprintf(w, "  Request %d.%d: %s %s\n", opNumber, j+1, req.Method, req.Path); err != nil {
				return fmt.Errorf("writing revert request summary: %w", err)
			}

			if req.Body != nil {
				bodyJSON, err := json.MarshalIndent(req.Body, "    ", "  ")
				if err != nil {
					return fmt.Errorf("marshaling body for revert operation %d request %d: %w", i, j, err)
				}

				if _, err := fmt.Fprintf(w, "    Body: %s\n", bodyJSON); err != nil {
					return fmt.Errorf("writing revert request body: %w", err)
				}
			}
		}
	}

	return nil
}
