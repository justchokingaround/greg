#!/bin/bash

# Read JSON input from stdin
INPUT=$(cat)

# Extract the command from JSON
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# If no command found, allow it (or handle as you see fit)
if [ -z "$COMMAND" ]; then
  exit 0
fi

# Define forbidden patterns customized for your Go project
# GOAL: Block high-token, low-value files (binaries, logs, images, lockfiles)
FORBIDDEN_PATTERNS=(
  # 1. Version Control & Metadata
  "\.git/"           # Entire git history (huge token waste)
  "\.claude/"        # Claude local storage

  # 2. Binaries (CRITICAL TO BLOCK)
  # Matches "greg" at the end of a string or followed by space/slash
  # Prevents 'cat greg' but allows 'cat internal/greg/file.go'
  "(^|/|\s)greg($|\s)"  
  "greg\.exe"        # Windows binary (19MB)
  "\.o$"             # Object files
  "\.test$"          # Go test binaries

  # 3. Media Files (Binary data = token explosion)
  "\.png$"
  "\.jpg$"
  "\.jpeg$"
  "\.gif$"           # tui.gif is ~800KB (huge)
  "\.ico$"

  # 4. Logs & Debugging (High noise, low value)
  "\.log$"           # ani-cli.log is 127KB
  "debug_output"     # Block debug_output.log
  "\.out$"

  # 5. Dependency Lock Files
  # go.sum is text, but usually irrelevant for coding logic 
  # and consumes lots of tokens. Only allow go.mod.
  "go\.sum"
)

# Check if command contains any forbidden patterns
for pattern in "${FORBIDDEN_PATTERNS[@]}"; do
  if echo "$COMMAND" | grep -qE "$pattern"; then
    echo "ERROR: Access to '$pattern' is blocked to save tokens." >&2
    exit 2
  fi
done

# Command is clean, allow it
exit 0
