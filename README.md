# immich-manager

A CLI tool for managing [Immich](https://immich.app/) albums through a
plan-based approach. Generate plans for album operations, review them, and then
apply or revert changes safely.

## Overview

immich-manager uses a plan/apply system:

1. **Plan Generation**: Create JSON plans describing what operations to perform
2. **Plan Execution**: Apply or revert the planned operations

This approach allows you to review changes before applying them and provides the
ability to undo operations.

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

```bash
# Replace text in album names
immich-manager plan albums replace [before] [after]

# Add user to albums matching search term
immich-manager plan albums add-user [search-term] [email]

# Remove user from all shared albums
immich-manager plan albums clear-shared [email]

# Sync smart album with contents of user's shared albums
immich-manager plan albums smart [email]

# Apply a plan
immich-manager apply [plan-file]
immich-manager apply --dry-run [plan-file]

# Revert a plan
immich-manager revert [plan-file]
immich-manager revert --dry-run [plan-file]
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
