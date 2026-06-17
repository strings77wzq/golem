#!/bin/bash
# CI Monitoring Loop — follows CLAUDE.md CI Monitoring Protocol
# Usage: ./scripts/ci-monitor.sh [max_checks] [interval_seconds]

set -euo pipefail

REPO="strings77wzq/golem"
MAX_CHECKS=${1:-10}
INTERVAL=${2:-90}

echo "=== CI Monitor Started ==="
echo "Repository: $REPO"
echo "Max checks: $MAX_CHECKS"
echo "Interval: ${INTERVAL}s"
echo ""

for i in $(seq 1 "$MAX_CHECKS"); do
    TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$TIMESTAMP] Check $i/$MAX_CHECKS"

    # Get latest run status using gh --json (no jq needed)
    RUN_JSON=$(gh run list --repo "$REPO" --limit 1 --json status,conclusion,databaseId,headBranch,createdAt 2>/dev/null || echo "")

    if [ -z "$RUN_JSON" ] || [ "$RUN_JSON" = "[]" ]; then
        echo "  ⚠ Network issue or no runs found, retrying in ${INTERVAL}s..."
        sleep "$INTERVAL"
        continue
    fi

    # Parse with grep/sed (no jq dependency)
    RUN_STATUS=$(echo "$RUN_JSON" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
    CONCLUSION=$(echo "$RUN_JSON" | grep -o '"conclusion":"[^"]*"' | head -1 | cut -d'"' -f4)
    RUN_ID=$(echo "$RUN_JSON" | grep -o '"databaseId":[0-9]*' | head -1 | cut -d':' -f2)
    BRANCH=$(echo "$RUN_JSON" | grep -o '"headBranch":"[^"]*"' | head -1 | cut -d'"' -f4)

    echo "  Status: $RUN_STATUS | Conclusion: $CONCLUSION | Branch: $BRANCH"

    # Case 1: CI passes
    if [ "$RUN_STATUS" = "completed" ] && [ "$CONCLUSION" = "success" ]; then
        echo ""
        echo "✅ CI PASSED — exiting loop"
        exit 0
    fi

    # Case 2: CI fails — investigate and fix
    if [ "$RUN_STATUS" = "completed" ] && [ "$CONCLUSION" = "failure" ]; then
        echo ""
        echo "❌ CI FAILED — investigating..."

        # Get failed jobs
        echo "  Failed jobs:"
        gh run view "$RUN_ID" --repo "$REPO" --json jobs 2>/dev/null | \
            grep -o '"name":"[^"]*"' | \
            grep -v '"name":"Set up"' | \
            grep -v '"name":"Run actions"' | \
            head -5 | \
            cut -d'"' -f4 | \
            sed 's/^/    - /'

        # Get failure logs (first 50 lines)
        echo ""
        echo "  Failure logs (first 50 lines):"
        gh run view "$RUN_ID" --repo "$REPO" --log-failed 2>/dev/null | head -50 | sed 's/^/    /'

        echo ""
        echo "📋 Action required: Fix the issue and push, then re-run this monitor."
        exit 1
    fi

    # Case 3: Still running
    echo "  ⏳ Still running, checking again in ${INTERVAL}s..."
    sleep "$INTERVAL"
done

echo ""
echo "⏰ Monitor timeout after $MAX_CHECKS checks"
exit 1
