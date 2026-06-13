# Spec: Tool Selector

## [S1] Tool Selection Algorithm

The `ToolSelector.Select` method returns the most relevant tools for a given step.

**Input:** step description, all available tools, max tools count
**Output:** []ToolDefinition (subset)

**Algorithm:**
1. Tokenize step description into keywords
2. For each tool, compute relevance score:
   - Name match: +3 if tool name contains a keyword
   - Description match: +1 per keyword found in description
   - ToolHints match: +5 if tool name is in step.ToolHints
3. Sort by score descending
4. Return top maxTools tools (default 8)

**Constraints:**
- Always include at least 1 tool (never return empty)
- Always include tool definitions (not just names) for LLM consumption
- Max tools default: 8 (configurable)

## [S2] Keyword Extraction

Extract meaningful keywords from step description:

**Rules:**
- Split on whitespace and punctuation
- Remove stop words (the, a, an, is, to, for, etc.)
- Lowercase all keywords
- Minimum keyword length: 2 characters

**Stop word list:**
```
the, a, an, is, are, was, were, be, been, being, have, has, had,
do, does, did, will, would, could, should, may, might, shall, can,
to, of, in, for, on, with, at, by, from, as, into, through, during,
before, after, above, below, between, out, off, over, under, again,
further, then, once, here, there, when, where, why, how, all, both,
each, few, more, most, other, some, such, no, nor, not, only, own,
same, so, than, too, very, just, because, but, and, if, or
```

## [S3] Fallback Behavior

When tool selection returns fewer than 2 tools:
1. Include all tools (don't filter)
2. Log a warning: "tool selection returned <2 tools, using all"

This ensures the LLM always has at least some tools available.

## [S4] Caching

Tool selection results are NOT cached between steps because:
- Different steps need different tools
- Tool registry may change (MCP tools added/removed)
- Simplicity over optimization in v1
