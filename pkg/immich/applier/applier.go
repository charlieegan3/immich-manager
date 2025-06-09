package applier

import (
	"fmt"
	"io"
	"encoding/json"

	"immich-manager/pkg/immich"
	"immich-manager/pkg/plan"
)

// Applier applies plans to the Immich API
type Applier struct {
	client *immich.Client
}

// ApplyOptions contains options for the Apply operation
type ApplyOptions struct {
	DryRun bool
	Writer io.Writer // Used for dry run output
}

// DefaultApplyOptions returns the default options for Apply
func DefaultApplyOptions() *ApplyOptions {
	return &ApplyOptions{
		DryRun: false,
		Writer: nil,
	}
}

// NewApplier creates a new plan applier
func NewApplier(client *immich.Client) *Applier {
	return &Applier{
		client: client,
	}
}

// Apply executes all operations in the plan
func (a *Applier) Apply(p *plan.Plan, opts *ApplyOptions) error {
	if opts == nil {
		opts = DefaultApplyOptions()
	}

	if opts.DryRun {
		return a.dryRunApply(p, opts.Writer)
	}

	for i, op := range p.Operations {
		req, err := a.client.NewRequest(op.Apply.Method, op.Apply.Path, op.Apply.Body)
		if err != nil {
			return fmt.Errorf("creating request for operation %d: %w", i, err)
		}

		if err := a.client.Do(req, nil); err != nil {
			return fmt.Errorf("executing operation %d: %w", i, err)
		}
	}

	return nil
}

// dryRunApply simulates applying the plan without making actual API calls
func (a *Applier) dryRunApply(p *plan.Plan, w io.Writer) error {
	if w == nil {
		return fmt.Errorf("writer is required for dry run")
	}

	fmt.Fprintf(w, "Dry run mode: would execute %d operations\n", len(p.Operations))
	for i, op := range p.Operations {
		fmt.Fprintf(w, "Operation %d: %s %s\n", i+1, op.Apply.Method, op.Apply.Path)
		if op.Apply.Body != nil {
			bodyJSON, err := json.MarshalIndent(op.Apply.Body, "  ", "  ")
			if err != nil {
				return fmt.Errorf("marshaling body for operation %d: %w", i, err)
			}
			fmt.Fprintf(w, "  Body: %s\n", bodyJSON)
		}
	}

	return nil
}

// Revert executes all revert operations in the plan in reverse order
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
		req, err := a.client.NewRequest(op.Revert.Method, op.Revert.Path, op.Revert.Body)
		if err != nil {
			return fmt.Errorf("creating revert request for operation %d: %w", i, err)
		}

		if err := a.client.Do(req, nil); err != nil {
			return fmt.Errorf("executing revert operation %d: %w", i, err)
		}
	}

	return nil
}

// dryRunRevert simulates reverting the plan without making actual API calls
func (a *Applier) dryRunRevert(p *plan.Plan, w io.Writer) error {
	if w == nil {
		return fmt.Errorf("writer is required for dry run")
	}

	fmt.Fprintf(w, "Dry run mode: would revert %d operations\n", len(p.Operations))
	for i := len(p.Operations) - 1; i >= 0; i-- {
		op := p.Operations[i]
		fmt.Fprintf(w, "Operation %d: %s %s\n", len(p.Operations)-i, op.Revert.Method, op.Revert.Path)
		if op.Revert.Body != nil {
			bodyJSON, err := json.MarshalIndent(op.Revert.Body, "  ", "  ")
			if err != nil {
				return fmt.Errorf("marshaling body for revert operation %d: %w", i, err)
			}
			fmt.Fprintf(w, "  Body: %s\n", bodyJSON)
		}
	}

	return nil
}
