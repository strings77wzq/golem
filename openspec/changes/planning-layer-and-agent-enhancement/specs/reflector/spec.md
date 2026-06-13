# Spec: Reflector

## [S1] Reflection Algorithm

The `Reflector.Evaluate` method checks whether a step achieved its goal.

**Input:** step (with ExpectedOut), execution result, error
**Output:** ReflectionResult

**ReflectionResult:**
```go
type ReflectionResult struct {
    Success   bool   `json:"success"`
    Reason    string `json:"reason"`
    ShouldRevise bool `json:"should_revise"`
}
```

**Evaluation heuristics:**
1. If error is non-empty → Success=false, ShouldRevise=true
2. If result is empty → Success=false, ShouldRevise=true
3. If result contains keywords from ExpectedOut → Success=true
4. If result length < 10 chars AND expected was substantial → Success=false
5. Default → Success=true (optimistic: assume success if no clear failure)

## [S2] Heuristic Keyword Matching

Compare result against ExpectedOut:

**Algorithm:**
1. Tokenize ExpectedOut into keywords (same rules as ToolSelector)
2. Count how many keywords appear in result
3. If >50% keywords match → Success=true
4. If 0% keywords match AND result is short → Success=false
5. Otherwise → Success=true (uncertain, assume success)

## [S3] Escalation to LLM

If heuristic evaluation is uncertain (result is long, keywords partially match):
1. Send to LLM: "Did this step achieve: {ExpectedOut}?\nResult: {result}\nAnswer: yes/no"
2. Parse LLM response
3. Use LLM's answer as final verdict

**When to escalate:**
- Result length > 500 chars
- Keyword match ratio between 20% and 50%
- Step is marked as "critical" (future extension)

## [S4] Reflection Integration

Reflection runs AFTER each step execution:

```
step execution → result → reflector.Evaluate() → 
  if success → continue to next step
  if failure → planner.Revise() → updated plan → continue
```

**Safety limit:** Maximum 3 revisions per plan. If plan is revised 3 times and still failing, mark plan as failed.
