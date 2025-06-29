# Immich Manager

A CLI tool for managing Immich albums through a plan-based approach. Generate plans for album operations, review them, and then apply or revert changes safely.

## Overview

Immich Manager uses a two-phase approach:
1. **Plan Generation**: Create JSON plans describing what operations to perform
2. **Plan Execution**: Apply or revert the planned operations

This approach allows you to review changes before applying them and provides the ability to undo operations.

## Installation

```bash
go build -o immich-manager ./main.go
```

## Environment Variables

The following environment variables are required:

- `IMMICH_TOKEN`: Your Immich API key
- `IMMICH_SERVER`: Your Immich server URL (e.g., `https://immich.yourdomain.com`)

You can set these in your shell:

```bash
export IMMICH_TOKEN="your-api-key-here"
export IMMICH_SERVER="https://immich.yourdomain.com"
```

## Commands

### Plan Generation

All plan generation commands output JSON to stdout. You can save plans to files for later execution:

```bash
immich-manager plan albums replace "old" "new" > plan.json
```

#### Album Operations

**Replace text in album names:**
```bash
immich-manager plan albums replace [before] [after]
```
- `before`: Text to find in album names
- `after`: Text to replace it with

Example:
```bash
immich-manager plan albums replace "2023" "2024"
```

**Add user to albums:**
```bash
immich-manager plan albums add-user [search-term] [email]
```
- `search-term`: Search term to match album names (case-insensitive substring match)
- `email`: Email address of the user to add to matching albums

Example:
```bash
immich-manager plan albums add-user "vacation" "user@example.com"
```

**Remove user from all shared albums:**
```bash
immich-manager plan albums clear-shared [email]
```
- `email`: Email address of the user to remove from all shared albums

Example:
```bash
immich-manager plan albums clear-shared "user@example.com"
```

**Smart album management:**
```bash
immich-manager plan albums smart [email]
```
- `email`: Email address of the user whose shared albums should be aggregated

This command manages a "smart album" that automatically contains all assets from albums shared with the specified user. The smart album must be named "All [User Name]" and must be created manually first.

Example:
```bash
immich-manager plan albums smart "user@example.com"
```

### Plan Execution

**Apply a plan:**
```bash
immich-manager apply [plan-file]
immich-manager apply --dry-run [plan-file]
```

- If no file is provided or `-` is used, reads from stdin
- `--dry-run`: Show what would be done without making changes

Examples:
```bash
# Apply from file
immich-manager apply plan.json

# Apply from stdin
cat plan.json | immich-manager apply

# Dry run to preview changes
immich-manager apply --dry-run plan.json
```

**Revert a plan:**
```bash
immich-manager revert [plan-file]
immich-manager revert --dry-run [plan-file]
```

- Undoes the operations from a previously applied plan
- `--dry-run`: Show what would be done without making changes

Examples:
```bash
# Revert changes
immich-manager revert plan.json

# Dry run to preview revert
immich-manager revert --dry-run plan.json
```

## Workflow Examples

### Bulk rename albums

```bash
# Generate plan to replace "2023" with "2024" in all album names
immich-manager plan albums replace "2023" "2024" > rename_plan.json

# Review the plan
cat rename_plan.json

# Apply the changes
immich-manager apply rename_plan.json

# If needed, revert the changes
immich-manager revert rename_plan.json
```

### Add user to vacation albums

```bash
# Generate plan to add user to all albums containing "vacation"
immich-manager plan albums add-user "vacation" "friend@example.com" > add_user_plan.json

# Preview what would happen
immich-manager apply --dry-run add_user_plan.json

# Apply the changes
immich-manager apply add_user_plan.json
```

### Maintain smart album

```bash
# Generate plan to sync smart album with shared albums
immich-manager plan albums smart "user@example.com" > smart_plan.json

# Apply the sync
immich-manager apply smart_plan.json
```

### Pipeline operations

```bash
# Generate and apply in one command
immich-manager plan albums replace "old" "new" | immich-manager apply

# With dry run
immich-manager plan albums clear-shared "user@example.com" | immich-manager apply --dry-run
```

## Smart Albums

Smart albums automatically aggregate all assets from albums shared with a specific user. To use this feature:

1. Create an album manually in Immich named "All [User Name]" (e.g., "All John Doe")
2. Share it with the target user
3. Run the smart album command to generate a sync plan
4. Apply the plan to sync assets

The smart album will:
- Add assets from any album shared with the user
- Remove assets that are no longer in shared albums
- Skip albums that start with "All " to avoid recursion

## Error Handling

- All commands validate required environment variables
- Plan generation will fail if users or albums are not found
- Apply operations provide detailed error messages with request/response information
- Dry run mode allows safe preview of all operations

## Development

Run tests:
```bash
go test ./...
```

Build:
```bash
go build -o immich-manager ./main.go
```

## License

This project is open source. Please check the license file for details.