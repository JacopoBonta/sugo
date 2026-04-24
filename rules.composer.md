---
id: pr-composer
title: Composer Team PR Guide
sidebar_label: Composer Team
slug: /guidelines/pr/composer
---

This guide introduces to team best practices related to git branching and pr.
Clear git history helps reviewers and developers; do your part for a better monorepo. 🙏

These guidelines are taking inspiration from [conventional commits specifications](https://www.conventionalcommits.org/en/v1.0.0/#summary) and [conventional branch](https://conventional-branch.github.io/#summary)

## How to name a branch

Changes always come from a Jira task. If you do not have one start by creating a jira issue.
The task of your jira issue identifies the type and provides a description to the work you are approaching
Branch will be named `<type>/<description>-<jira issue>`
Possible type are:

- `feat/`: for new features (e.g.`feat/add-login-page-com-xxx`)
- `fix/`: for bug fixes (e.g. `fix/header-bug-com-xxx`)
- `hotfix/`: For urgent fixes over specific release branches (e.g., `hotfix/security-patch-com-xxx`)
- `release/`: For branches preparing a release (e.g., `release/v1.2.0`)
- `chore/`: For non-code tasks like dependency, docs updates (e.g., `chore/update-dependencies-com-xxx`)

## How to write concise and useful commit messages

Git history help other developer in review and rebase when the changes are too much and is required to check the changes one by one.

To communicate the intent of the commit the following prefix are used:

- `fix`: a commit of the type fix patches a bug in your codebase (this correlates with PATCH in Semantic Versioning)
- `feat`: a commit of the type feat introduces a new feature to the codebase (this correlates with MINOR in Semantic Versioning)
- `feat!` or `fix!`: a commit that has a footer `BREAKING CHANGE:`, or appends a `!` after the type/scope, introduces a breaking API change (correlating with MAJOR in Semantic Versioning)
- Additional types: are not mandated by the Conventional Commits specification, and have no implicit effect in Semantic Versioning (unless they include a BREAKING CHANGE). They are: `build`, `chore`, `ci`, `docs`, `style`, `refactor`, `perf`, `test`

Every commit can be provided with a scope and add more contextual information (e.g. `fix(iam): ....`)
_Introducing too many types would be unnecessary and would make them harder to manage and remember._

To better track jira issue in the codebase and facilitate complex rebases inside feature branch, we require to insert the jira issue id in tail of the commit message.

## How to title GitHub PRs

- PR representing a single task against develop should be merged using squash and merge.
- PR representing epic from feature branch (aggregation of multiple PRs) should be merged against develop using rebase.
- PR representing a single task against a feature branch should be merged using squash and merge.
In case of rebase the commit will be ported to the destination branch, in case or squad and merge the title of the pr will become the commit message on the target branch.
_Squash and merge allow to edit the commit message but that is not possible if the pr get merged automatically by the bot_
For that reason the PR title is very important for clean git history.
**The solution is to name PR as the commit.**
e.g `feat(composer): add support for operator login and redirect to previous url com-xxx`

## How to describe the pr correctly

Github copilot is really useful to draft pr description mentioning all the changes.
**Please read them and check against jira description before submit.**

If you are preparing the merge of a single task use one of `.github/PULL_REQUEST_TEMPLATE/COMPOSER_TEAM_CLIENT.md` or `.github/PULL_REQUEST_TEMPLATE/COMPOSER_TEAM_SERVICE.md` or `.github/PULL_REQUEST_TEMPLATE/COMPOSER_TEAM_OPERATORS.md`.
If you are preparing the merge of a feature branch (multiple jira issue grouped in the same git branch covering multiple services and clients) is mandatory to use `.github/PULL_REQUEST_TEMPLATE/COMPOSER_TEAM_FEATURE_BRANCH.md`

The pr template will drive you in redacting a comprehensive description for your colleague.

## How to select the correct labels in pr

### Automatic merge

`bot/merge` and `bot/skip` controls the automatic merge of the pr by cubbot.
Automatic merge take place only when review policy are satisfied and test are all green.

### Priority

Labels can be use to notify the urgency of a pr.
Reviewer should prioritize reviews over pr that are more urgent.

These labels all start with `priority/...`

### Identify Services and Client

Labels like `section/...` are automatically added by the bot based on the file you have touched.
These labels are really useful to find pr on specific areas of the monorepo.

The same results can be reached looking at the git history of the folder you are interested in 🕵.

### Migrations

Migrations label must be added manually and carefully.
If your changes are introducing any kind of migration add the `migration` label.
**Remember to include in the pr the release note instruction or the pr will be closed.**

Then on top of that add one of the following label to specify the type of action that is required:

- `migration/k8s`: for new K8S secrets, configmap, and other K8S resources
- `migration/kafka-topic`: for new Kafka topics
- `migration/DB`: for a DB schema update

## How to ensure nothing will break with the introduced changes

Test are automatically executed on you pr, so add them to the codebase to increase the coverage.

E2E tests are heavy and need to be triggered manually on the specific branch. That action is suggested on all the branch that may impact other teams or are adding new feature to the target branch.
_Remember to re-run test after rebase operations_

Tests are available in the [monorepo github actions](https://github.com/cubbit/cubbit/actions)
Please add the test status banner in pr to facilitate the check of the results. The banner is associated to the branch so does not need to be updated after every new execution of the tests.

## You have reached the end of the document

If something is missing, feel free to provide suggestion and integration to the current document.
