---
name: feature-pr
description: >-
  Feature-wise git branch, push, and GitHub PR for any Facinect microservice or
  app (identity, notifications, web, gateway, and future services). Use when the
  user asks to ship a feature, create a PR, push service changes, open a pull
  request, feature branch, or feature-wise code push.
---

# Feature-wise branch + PR (all services)

Monorepo workflow for **every** service/app — not notifications-only.

## Scope

| Area | Paths |
|------|--------|
| Services | `services/*` (identity, notifications, future `booking`, etc.) |
| Apps | `apps/*` (web, …) |
| Platform | `gateway/`, `docker-compose*.yml`, `docs/`, `scripts/`, `sql/` |

Never commit secrets: `.env`, `.env.prod`, keys, tokens, credential dumps.

## Triggers

`ship feature`, `feature PR`, `push this service`, `create PR`, `feature branch`, `feature-wise push`.

## Workflow

1. **Inspect** (parallel): `git status`, `git diff` (staged + unstaged), `git log -5 --oneline`, `git branch -vv`, remote tracking.
2. **Map feature**:
   - Primary path under `services/<name>/` → service = `<name>`
   - Else `apps/<name>/` → service = `<name>`
   - Else platform-only → service = `platform`
   - Slug: 2–4 kebab words from the change intent (e.g. `facility-wa`, `docs-spec-picker`)
3. **Branch**: `feat/<service>-<slug>`
   - Base: `main` unless user names another base
   - `git fetch origin`
   - If branch missing locally and on `origin` → create from `origin/<base>` (or local base)
   - If exists remotely → checkout / track it; apply current work onto it
   - Do not force-push `main`/`master`
4. **Stage** only files for this feature (exclude secrets and unrelated junk).
5. **Commit** when user asked to ship/PR (that implies commit). Message: why-focused, 1–2 sentences. Use HEREDOC. Follow repo commit style from `git log`.
6. **Push**: `git push -u origin HEAD` (needs network/`all` permissions as required).
7. **PR** with `gh pr create`:
   - Title: `<service>: <short why>`
   - Body HEREDOC:

```markdown
## Summary
- …

## Test plan
- [ ] …
```

8. **Reply** with the PR URL. Stop.

## Multi-service one feature

One branch + one PR is OK. Title: primary service; mention `+ shared` if compose/docs/gateway also changed. Split into multiple PRs only if the user asks.

## Model note

Works with any Cursor agent model (including Grok). Skill logic is model-agnostic.

## Examples

- Notifications docs picker → `feat/web-docs-spec-picker` → PR title `web: add Service picker on /docs`
- Facility WhatsApp hybrid → `feat/notifications-facility-wa`
- New `services/booking` MVP → `feat/booking-mvp-api`
- Traefik file-provider only → `feat/platform-traefik-file-routes`
