# Final review of Lesson 13 (raw output)

Reviewer: a fresh Claude Code subagent (general-purpose, Claude Fable 5.1), read-only on the checkout, prompt in
`prompts.txt` (section "Final review"). Range `7f6cbf2..1cdbc85`, 2026-10-27. The report is reproduced unedited;
the triage is in `README.md`, section "Lesson 13 Report: gRPC Microservices", subsection "Фінальне рев'ю".

---

Review of Lesson 13 (albums), range 7f6cbf2..1cdbc85 (HEAD of main), checkout untouched (`git status` clean before and after; the CI static replay ran in a scratchpad clone so `make proto`/`go mod tidy` never wrote into the repo). Repo root R = /Users/oleksandr.tatarchuk/repo/golang-course-lesson-13.

### Strengths
- Report accuracy: every number, code, message and file name I could trace in R/README.md matched its evidence file or the code: environment table (sw_vers 27.0.1/26A434, go1.27.1, libprotoc 36.2 vs header v7.36.2, plugins v1.34.2/1.5.1/v2.20.0, grpcurl v1.9.4, demo go.mod grpc v1.64.1/protobuf v1.34.2/gateway v2.20.0, own grpc v1.85.0/protobuf v1.36.12); the CI claims (ubuntu-26.04, protoc 3.21.12, Go per go.mod) are confirmed by the job logs of run 40576869474 (Image: ubuntu-26.04, libprotoc 3.21.12, go1.27.3); go-test.txt (18-line go.sum, three ok packages); quickstart/unary/server-stream (3 updates, `reason="max_updates reached" sent=3`, Ctrl+C → `client gone / cancelled`, `code=Canceled`, -simulate 5 messages in 7.718975 s at 2 s intervals); client-stream (5/0 then 0/5, five WARN lines), bidi (true/true/false, "rejected: copy not found"); gateway (200/404 code 5/POST copy-4/409 code 6/PATCH LOST/400 code 3, three NDJSON lines); auth (all codes, 401 with Www-Authenticate, 200, 403, masked tokens); mtls (nine PEMs, the four negatives, the two info-level reasons); tests/bench (28 B vs 85 B 3.0x; 60653/49537/125921 ns; 16145/15366; 215.6/637.7 ns; README-demo reference numbers 410/1850 ns, 33/24/80 us at demo README:316-320); smoke (9 OK + ВСЕ ГАРАЗД); proto-regen (2 files x 1 line, v7.36.2 vs v3.21.12); own service: 35 tests = 20+4+11, coverage 100/86.3/96.2/30.0, 7 unary + 3 streaming smoke checks, every grpcurl/client code and message, both `watch finished` reasons, the WARN text, port 50061, buffer 16, first new id 11.
- Course demo is verbatim: `diff -r` against ~/Downloads/libratrack-grpc-demo.zip differs only in R/libratrack-grpc-demo/go.mod (the 4 indirect requires) and the added go.sum.
- Own service meets the task point by point: three rpcs, exactly one `stream`; InvalidArgument/NotFound/FailedPrecondition/AlreadyExists/Internal mapped in R/internal/server/server.go:48-122; no errors.New/fmt.Errorf there (test + CI + my grep); WatchAlbums uses stream.Context(), subscribes before streaming, returns nil after max_events, status.FromContextError on cancel, `defer cancel()` on every exit; plaintext with a WARN (R/cmd/albums-server/main.go:58); reflection registered; client get/create/watch with protojson, `code=... message=...`, io.EOF and Ctrl+C handling.
- Store concurrency is sound: publish happens under the write lock and cancel closes the channel under the same lock (no send on closed channel), `sync.Once` makes cancel idempotent, `done` stops the per-subscription goroutine, sends are non-blocking (R/internal/store/store.go:128-168). Validation texts and seed data are byte-for-byte the Lesson 12 ones (checked against ~/repo/golang-course-lesson-12/internal/model/album.go, validate.go and migrations/000002_seed_demo_data.up.sql, including album_artists links and album 10's NULL year).
- Tests run over a real grpc.Server on 127.0.0.1:0 with the production interceptor chain and wait on SubscriberCount() instead of sleeping. tests-before.txt and tests-after.txt describe the same test set: the `wc -l` figures (317/65/264) equal the committed files and every compile-error position in the red log (store_test.go:14:19 AlbumInput, 17:9, 27:28, 50:7, 63:42, 69:12, 76:17, 91:12, 101:7, 103:43; middleware_test.go:29:12, 37:9, 47:9, 59:6) lands on the same identifier in the current files.
- CI does what the README says; the generated-code check tolerates only the protoc header line and passed on Ubuntu's protoc 3.21.12; no README link or CI required file was ever named before it existed in any of the 10 commits; author is the gmail address on all commits, no attribution lines. HEAD is pushed (remote main = 1cdbc85) and its run 40580669193 is green.

### Issues

#### Critical (Must Fix)
None.

#### Important (Should Fix)
1. R/internal/store/store.go:142-148 together with R/internal/server/server.go:87-91 and R/internal/server/server_test.go:236-250 — the per-subscription context goroutine duplicates the handler's `defer cancel()` and causes two problems. (a) The leak test does not prove the handler loop ended: `SubscriberCount()` drops to 0 through the store's goroutine on ctx.Done(), not through the handler's defer, so the test passes even if WatchAlbums never returns. Evidence: CI run 40576869474 reports internal/server coverage 83.9% while 20 local runs (default, -cpu 1, -cpu 2) all report 86.3% with an identical uncovered set; the 2.4% gap is exactly the two statements of the ctx.Done() branch (server.go:84-86, 2 of 73 statements), i.e. in CI the test finished before the handler's exit statements ran. The README (line 300-303) cites this test as the slide-46 proof. (b) When a cancel is observed after the goroutine has already closed the channel, select may take the `!ok` branch and return nil, so the interceptor logs code=OK for a cancelled stream instead of Canceled (client unaffected). Fix (one change covers both): either drop `ctx` from `Subscribe` and the goroutine (the handler owns the lifetime; then `SubscriberCount()==0` can only come from the handler's defer and the `!ok` branch is dead and can go), or keep it and (1) in the `!ok` branch return `status.FromContextError(ctx.Err()).Err()` when `ctx.Err() != nil`, (2) in `start()` log to a bytes.Buffer and `waitFor` the `msg="watch finished"`/`code=Canceled` line in the cancel test. The README store paragraph (line 231-233) would need a one-line update if the ctx variant goes.

#### Minor (Nice to Have)
2. R/README.md:172-173 — "чотири негативні сценарії дають те, що описано в README демо" is not exact: for `rogue-client` the demo README (R/libratrack-grpc-demo/README.md:274) expects `connection reset by peer`, macOS gave `tls: unknown certificate authority` (R/docs/demo/mtls.txt:74). Likewise the demo README:107 expects `addr=0.0.0.0:50051`, the Mac logs `[::]:50051`. Say "three as written, rogue-client with a more specific alert" and/or add the row to the deviations table (README:196-201).
3. R/prompts.txt:6-7 — names R/docs/review-raw.md and a README subsection «Фінальне рев'ю»; neither exists at HEAD (the only forward references in the range). They are planned for the review commit (plan 5.3); make sure that commit adds both, or reword.
4. R/albums_lesson13_homework.md:70,79,108 (EN 182,191,220) — still promises screenshots S13–S16, S17–S21 and S1–S21 while the README delivers S1–S14 and only explains the Task 3 drop (README:279-281). Add one sentence about the four-to-two screenshot ruling (progress.md ruling of 2026-10-27).
5. R/.github/workflows/lesson13-check.yml:5 and :156 — stale Sonar references after the drop: the header comment lists a `sonar` job and the upload step is named "Keep the coverage profile for the SonarQube Cloud job". Rename ("as a run artifact") and drop `sonar` from the comment.
6. R/.github/workflows/lesson13-check.yml:107 — `ascii_only` needs `grep -P`. Under bash on this Mac `grep` is BSD grep 2.6.0 (no -P): grep exits 2 with its usage text and the `!` turns that into a PASS, so a local replay never checks ASCII (my replay printed PASS for that reason; the zsh session uses ugrep, which hides it). CI's GNU grep is fine and a portable `LC_ALL=C grep -rn $'[\x80-\xFF]' cmd internal gen` finds nothing. Make the check portable (that pattern) or guard: `grep -P '' /dev/null 2>/dev/null || return 1`. The unquoted `--include=*.go` is also a zsh trap for local replays.
7. R/internal/server/server.go:53-55 and :116-117 — ErrNotFound is mapped twice; the `toStatus` case is unreachable (line 117 never covered). Keep one.
8. Test gaps worth a test: (a) the Internal path — `toStatus` default (server.go:118-120) and GetAlbum's generic error branch (57-58) are never executed; a fake Catalog returning an arbitrary error should yield codes.Internal with the generic message only; (b) the notes round trip over gRPC — R/internal/server/convert.go:47-50 is never covered by tests (only by the manual logs); (c) unknown artist → FailedPrecondition at the gRPC level (only the store level tests it); (d) R/cmd/albums-client/main.go `createRequest` parsing ("1,2", bad ids, -year/-notes omission) has 0% coverage.
9. R/README.md:255 — "чотири повідомлення валідації": the table test (server_test.go:170-179) has five cases over four field names. Wording.
10. R/README.md:198 — the DeadlineExceeded row cites `bin/server.log` (git-ignored) and DNS timings that are in no evidence file; the only report claim without one. Say "not captured" or add a short note under docs/.
11. Screenshots: R/docs/screenshots/s01-versions.png shows `grpcurl -version` as "dev build <no version set>" while the S1 row says it shows grpcurl's version (v1.9.4 is only in environment.txt via `go version -m`); R/docs/screenshots/s13-albums-run.png contains a mistyped command (`zsh: permission denied: ./cmd/albums-client`) and only the create terminal (the row does say events/EOF are in the log). Optional retake. All other 12 screenshots match their rows.
12. R/cmd/albums-client/main.go:39,133 — a negative `-year` is silently treated as "omitted" instead of being sent for validation; negative `-max` becomes unlimited; `watch` ignores extra arguments. Nits.
13. R/cmd/albums-server/main.go:42-43 with R/internal/middleware/middleware.go — recovery outermost means a panicked call gets no `rpc` log line (only "panic recovered"); logging outermost would record code=Internal for it.
14. R/.github/workflows/lesson13-check.yml:24,26,138,140,157,169,171 — checkout@v4, setup-go@v5, upload-artifact@v4 run on a forced Node 24 with deprecation annotations in every job; bump to the Node-24 majors when convenient.
15. R/internal/store/store.go:159-168 — drop-when-slow is silent: a lagging watcher misses events and, with max_events, waits for events it will never count. Document in the proto comment or send a lag marker. Design note, documented policy.

### Declined to judge
- Task 3 (SonarQube Cloud) not done — user ruling of 2026-10-27; the README states it.
- Only two own-service screenshots (S13, S14) — user ruling "3-5 screenshots"; judged only the doc inconsistency (item 4).
- No separate red commit for tests-before.txt — plan 2.2 defines the log as the proof.
- Duplicate (title, label) check is exact-match (case-sensitive, untrimmed) — the task asks for the Lesson 12 rules, which are the same family.
- O(n) duplicate scan under the write lock — in-memory demo, no scale requirement.
- Negative max_events treated as unlimited — the contract defines only 0 and > 0.
- No TLS, auth or gateway for the own service — plan's out-of-scope list.
- Server hardening (keepalive, MaxConcurrentStreams, message limits) — local development only per the task.
- README CI links point to runs of 612d125 and ecf6a12; the green HEAD run 40580669193 is to be linked in plan step 5.4.
- The demo's own code quality — course-supplied, reviewed only for intactness.
- `go.mod` has no `toolchain` line — not required; CI resolved go1.27.3.

### Command output (step 5, the verbatim tails)
Root (R), `gofmt -l .` → no output, exit 0. `go vet ./...` → no output, exit 0.
`go test -race -count=1 ./...` (exit 0):
```
?   	homework/cmd/albums-client	[no test files]
?   	homework/cmd/albums-server	[no test files]
?   	homework/gen/albums	[no test files]
ok  	homework/internal/middleware	1.461s
ok  	homework/internal/server	1.510s
ok  	homework/internal/store	1.373s
```
`bash scripts/smoke.sh` (exit 0):
```
unary:
  OK   GetAlbum 1 returns Kind of Blue
  OK   GetAlbum 999 -> NotFound
  OK   GetAlbum 0 -> InvalidArgument
  OK   CreateAlbum -> id 11
  OK   CreateAlbum duplicate -> AlreadyExists
  OK   CreateAlbum unknown genre -> FailedPrecondition
  OK   CreateAlbum without title -> InvalidArgument
server streaming:
  OK   WatchAlbums delivers the 2 CREATED events
  OK   WatchAlbums ends with io.EOF after max_events
  OK   server log: watch finished, max_events reached
ALL OK
```
R/libratrack-grpc-demo: `go vet ./...` → no output, exit 0. `go test -race -count=1 ./...` (exit 0):
```
ok  	libratrack/bench	1.486s
?   	libratrack/cmd/gateway	[no test files]
?   	libratrack/cmd/gencerts	[no test files]
?   	libratrack/cmd/inventory-client	[no test files]
?   	libratrack/cmd/inventory-server	[no test files]
?   	libratrack/cmd/mint-token	[no test files]
?   	libratrack/gen/inventory	[no test files]
ok  	libratrack/internal/auth	1.456s
?   	libratrack/internal/middleware	[no test files]
ok  	libratrack/internal/server	1.499s
?   	libratrack/internal/store	[no test files]
?   	libratrack/internal/tlsconf	[no test files]
```
`bash scripts/smoke.sh` (exit 0):
```
gRPC:
  OK   GetCopy повертає Copy
  OK   GetCopy невідомий -> NotFound
  OK   Register дубль -> AlreadyExists
  OK   Bulk реєструє партію
  OK   Sync підтверджує події
  OK   Watch завершується io.EOF
REST (gateway):
  OK   GET /v1/copies/copy-1 -> 200
  OK   GET невідомий -> 404
  OK   POST дубль -> 409
ВСЕ ГАРАЗД
```
Static job replay (run blocks extracted with `ruby -ryaml` from the HEAD workflow, executed with bash in a scratchpad clone of HEAD at /private/tmp/claude-501/-Users-oleksandr-tatarchuk-repo-golang-course-lesson-13/38b82f35-3e47-4037-8a34-3f938f2bc0db/scratchpad/review-clone, RUNNER_TEMP and GITHUB_STEP_SUMMARY under that scratchpad; the apt/make-tools step skipped, local tools libprotoc 36.2, protoc-gen-go v1.34.2, protoc-gen-go-grpc 1.5.1; the clone was still clean after `make proto` and `go mod tidy`):
```
PASS: Required files exist
PASS: Module name is homework
PASS: No .go files in the repository root
PASS: gofmt: all files formatted
PASS: go vet ./... (root module)
PASS: go build ./... (root module)
PASS: go mod tidy leaves go.mod/go.sum unchanged
PASS: No non-ASCII characters in the homework's .go files   (vacuous on BSD grep, see item 6; CI and a portable grep agree there is none)
PASS: No JWT committed under docs/, in the README or in prompts.txt
PASS: .env is ignored and not committed
PASS: Shell scripts parse (bash -n)
PASS: Server handlers use status errors only
PASS: proto: package albums.v1 (versioned package, slide 17)
PASS: proto: three rpcs
PASS: proto: exactly one server-streaming rpc
PASS: Generated code is current (make proto)
```
16 PASS, 0 FAIL; step exit 0. CI itself: the same 16 PASS lines in run 40576869474 (15 there, the prompts.txt guard was added in HEAD) and run 40580669193 green for HEAD.

### Recommendations
- Do item 1 as one change (preferably remove the ctx goroutine from Subscribe; otherwise the ctx.Err() check plus a server-side assertion in the cancel test); add the Internal-path test (8a) at the same time since it reuses the Catalog interface.
- In the review commit: add R/docs/review-raw.md and the «Фінальне рев'ю» subsection that prompts.txt already names, link run 40580669193, mention the rogue-client difference and the screenshot ruling, rename the two stale Sonar strings in the workflow.
- Make `ascii_only` portable so local replays mean what CI means.

### Assessment
Ready to submit? With fixes. Every checked requirement of Tasks 1 and 2 is met and the report's numbers, codes and messages are backed by the evidence files, so nothing blocks submission; the one Important item is a test that proves less than the README says it does (and a related logging nondeterminism), and the rest are documentation precision and stale names that the planned review commit can clear in one pass.
