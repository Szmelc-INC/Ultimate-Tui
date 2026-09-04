# OPTIMAL ECO MODE — Balanced Token Ruleset

## 1. Surgical Reconnaissance (Highest Input ROI)
Never read blindly. Use native CLI tools to pinpoint context before ingestion.
- **Search:** `rg -n "KEYWORD" -C3` (preferred) or `grep -rn "KEYWORD" .`
- **Structure:** `head -n 40 file` or `grep -E "^(class|def|function)" file`.
- **Targeted extraction:** `sed -n 'START,ENDp' file`.
- **Hard Limit:** Never `cat` a file > 250 lines purely for "context". Only ingest full files if you are actively rewriting their core logic.

## 2. Model & Compute Routing
- **Haiku:** Recon, log parsing, grepping, formatting, single-line mechanical edits.
- **Sonnet:** Active coding, cross-file reasoning, debugging, architecture.
- **Effort/Thinking:** Lock at `low` for 90% of tasks. Escalate to `medium` ONLY if writing complex logic or if a previous `low` attempt failed. No extended thinking.

## 3. Tool & Context Hygiene (Zero-Overhead State)
- **Drop Subagents:** Do NOT spawn subagents for codebase exploration (the initialization token overhead outweighs the context savings). Use `rg` and `fd` instead.
- **Truncate Noise:** If a bash command yields massive output (e.g., long logs, massive JSONs), pipeline it immediately: `| head -n 30` or `| jq '.summary'`.
- **No Echoes:** Never output/echo raw stack traces, massive tool outputs, or unchanged code snippets back to the context.

## 4. Output Muting (Highest Output ROI)
- Zero conversational filler. No "Here is the plan", "I will now", or "Here is what I found."
- Answer in terminal-style outputs (1-2 lines). 
- If asked a question, provide the answer directly without repeating the premise.
- Announce model switches in 3 words or less (e.g., `[Sonnet: active coding]`).

## 5. The Performance Escape Hatch
Frugality must not cause infinite retry loops (which burn more tokens than doing it right the first time). If a refactor genuinely requires whole-file context, or if a bug involves deep stack traces, state `[Bypass: requires deep context]`, execute the expensive read, and solve it in one shot using Sonnet.
