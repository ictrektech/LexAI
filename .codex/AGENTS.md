# VAI Agent Instructions

These repository-wide instructions apply to AI coding agents working in this checkout.

## Build And Test Location

- Before running any build, test, packaging, release, CI-reproduction, or verification command, check whether the user has explicitly specified where it should run.
- If the location is not explicit, ask the user which remote server/environment to use before running the command.
- Do not assume local execution, a default remote host, or a previous remote server unless the user explicitly says to reuse it in the current task.
- This applies to commands such as `pnpm build`, `npm test`, `cargo test`, `pytest`, `make`, `docker build`, `docker compose`, Tauri builds, frontend test runners, and project-specific CI scripts.
- When testing local code on a remote server, do not run `git` commands on the server. Copy the local working tree or required files to the server through `ssh`, `rsync`, `scp`, or another available connection channel, then run the requested build/test commands there.
- It is still fine to run read-only inspection commands locally, such as `ls`, `rg`, `git status`, `sed`, and `find`.


## Feishu Release Table Lookup

- Some component image tags are recorded in a shared Feishu spreadsheet used by build scripts such as `build_image.sh`.
- Feishu app credentials may exist on build/deployment hosts at `~/.feishu.json`; do not print or copy the credential values into the conversation or repository.
- The release spreadsheet token currently used by local build scripts is `Htotsn3oahO1zxt73YMcaB1zn8e`.
- To find an image tag, use the target component column in the relevant sheet instead of inferring from a nearby component. Common sheets include `AMD_with_cuda`, `AMD_with_mxn100`, `ARM_with_cuda`, `ARM_without_cuda`, `l4t`, `thor_spark`, and `SOPHON_bm1688`.
- When `build_image.sh` builds an AMD Docker image, update both `AMD_with_cuda` and `AMD_with_mxn100`. When it builds an ARM image, update `ARM_with_cuda`, `ARM_without_cuda`, `l4t`, `thor_spark`, and `SOPHON_bm1688`.
- In general, use `ssh tc232` for AMD platform builds/tests and `ssh tc192` for ARM platform builds/tests.
- Spreadsheet URL pattern: `https://*.feishu.cn/sheets/Htotsn3oahO1zxt73YMcaB1zn8e`.
- Build-host credential file: `~/.feishu.json`, with `feishu_app_id` and `feishu_app_secret`.
- The release table uses one service image per column. Row 1 is the service name, row 2 is the SWR repository URI, and dated rows contain image tags. The full pullable image is `<row-2-repository-uri>:<dated-row-tag>`.
- Treat Feishu as the tag source of record, then verify pullability with `docker manifest inspect <image>` on the target host before reporting an image as usable.

## README Updates

- When adding a new feature, changing behavior, or changing how users/operators/developers should invoke or configure something, update the relevant submodule README in the same task.
- Document the practical usage difference: commands, configuration keys, environment variables, API behavior, deployment steps, migration notes, or screenshots/examples when they are part of the changed workflow.
- If no appropriate README exists, either create one in the affected submodule or explicitly tell the user why no README update was made.

## Evidence Before Claims

- Do not guess runtime causes from symptoms when a live repository, database, queue, log, or container state can answer the question.
- Before saying a deployment, task, queue, model, feature flag, or UI behavior is working, inspect the current target state and report the concrete evidence used.
- If evidence is not available yet, say that it is not verified instead of presenting an inference as fact.

## Instruction Maintenance

- Keep this `.codex/` directory tracked in git. Do not add it to `.gitignore`.
- Update these instructions when repository workflow expectations change.
