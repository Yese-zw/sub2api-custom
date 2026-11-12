# Parity fixtures

`TestParityWithNewAPIJudge` requires this package to reproduce, bit for bit,
the outputs of the original judge (the built-in judge in our New API service,
`service/channel_intelligence_judge.go`) on the same inputs.

| File | Content |
|---|---|
| `cases.jsonl` | 523 synthetic inputs `{case, prompt, html \| html_b64}`; no real model output |
| `newapi-recorded.jsonl` | Outputs of the original judge on each case: `quality`, `borderline`, `method`, `score`, `margin`, `reason` |
| `gen/main.go` | Deterministic generator of `cases.jsonl` |
| `recorder/record_newapi_test.go.txt` | Test file that produced `newapi-recorded.jsonl` inside the original service |

Inputs: the three demo samples under a bank prompt, an off-bank prompt and a
bank prompt with a trailing space; one seeded blend of normal-style lines and
degraded-style tokens for each of the 200 bank prompts, judged both in and out
of the bank; 17 text-normalization edge cases (empty, CRLF, NBSP runs, ASCII
information separators, U+0130, combining marks, invalid UTF-8, ...) in and
out of the bank; 80 random strings.

Recorded on 2026-11-10 from the original service at commit `4512ef62b67e`
with the pinned assets (`model.json` `fc07b7cc…`, `references.json` `2355c003…`).

Regenerate the inputs (from the module root; output must be unchanged):

```bash
go run ./judge/testdata/parity/gen > judge/testdata/parity/cases.jsonl
```

Re-recording needs the original service source; see the header of
`recorder/record_newapi_test.go.txt`.
