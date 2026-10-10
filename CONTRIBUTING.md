# Contributing

How changes are made in this repository. The same rules apply in every Arcline
repository.

## The flow

```
your-branch ──pull request──▶ dev ──▶ qa ──▶ staging ──▶ production
```

- `dev` is the source of truth. All work merges into it.
- `qa`, `staging` and `production` only ever receive what the stage before
  them has. Nobody commits to them directly.
- `main` is not part of the flow. Do not branch from it or target it.

## Making a change

1. Start from an up-to-date `dev`: `git switch dev && git pull --ff-only`.
2. Create a branch named for the change, in lowercase with hyphens:
   `add-dkim-rotation`, `fix-ticket-reply-email`. No names, dates or ticket
   numbers.
3. Keep it small and short-lived: one purpose, merged within a day or two.
   Split anything bigger, and hide unfinished work behind a flag or by not
   linking to it.
4. Before pushing: `bin/rails test`, `bin/rubocop` and `bin/brakeman` pass.
   `bin/ci` runs all of them.
5. Push and open a pull request into `dev`.

## Commits

```
Add the DNS check

The domain page listed the records to publish but could not say whether
they were live. This compares them with what DNS actually answers.
```

- **Subject:** what the commit does, in the imperative ("Add", "Fix",
  "Remove"), sentence case, no full stop, about 60 characters at most.
- **Body, when the subject is not enough:** why the change was made and
  anything surprising about it, wrapped at about 72 characters. Say what you
  could not test.
- **One purpose per commit.** A fix and a tidy-up are two commits.
- No type prefixes (`feat:`, `fix:`), no ticket numbers in the subject, no
  co-author trailers.
- Commits made by automation keep their own form, such as
  `build: automatic update of <app>`, so they are easy to tell apart.

## Pull requests

- **Title:** written like a commit subject. It becomes part of the history.
- **Description:** fill in the template. "Not checked" matters as much as
  "How it was checked".
- **Size:** something a reviewer can read in about ten minutes.
- **Review:** one approval from someone other than the author. The review bot
  comments on every pull request; that is a second pair of eyes, not an
  approval. Changes to authentication, encryption or anything handling
  secrets need a reviewer who knows that code.
- **Green before merge:** nothing merges with failing checks.
- **Merging:** use a merge commit, then delete the branch.
- **After merging:** if something goes wrong, revert on `dev` with a new
  commit. Never rewrite the history of `dev`, `qa`, `staging` or `production`.

The review bot sends each pull request's diff to an outside service. Never put
a secret in a diff, even one you intend to remove in a later commit.

## Promotion

Moving code from one stage to the next is a fast-forward and adds no commits,
so it has no pull request description to write:

```bash
git fetch origin
git push origin origin/dev:qa
```

If git refuses, someone committed to the stage directly. Do not force it.

## Reporting a problem

Open an issue using the bug report template. Leave out secrets and customer
data. For a security problem, contact the maintainer directly instead of
opening an issue.
