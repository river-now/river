#

## IMPORTANT, ALWAYS RUN A BUILD FIRST

```sh
pnpm build
```

**_^^^^^^^^^^^^ Do not forget this please ^^^^^^^^^^^^_**

# Pre-release workflow

https://github.com/changesets/changesets/blob/main/docs/prereleases.md

```sh
pnpm build
pnpm changeset pre enter beta
pnpm changeset version
pnpm changeset publish
git add .
git commit -am "whatever"
git push --follow-tags
```

# Normal release workflow

## Whenever you make a change that should be noted in a release changelog, add a changeset

```sh
pnpm changeset
```

## Whenever you want to release

1. Run a build!

```sh
pnpm build
```

2. Bump versions

```sh
pnpm changeset version
```

Then REVIEW THE CHANGESETS

DO NOT COMMIT ANYTHING BETWEEN CALLED `version` and `publish`!

Then,

```sh
pnpm changeset publish
```

Now, once published, it's finally OK to commit. Include `--follow-tags` to push the tags.

```sh
git push --follow-tags
```
