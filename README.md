# Gatekeeper

[![License](https://img.shields.io/github/license/FollowTheProcess/gatekeeper)](https://github.com/FollowTheProcess/gatekeeper)
[![GitHub](https://img.shields.io/github/v/release/FollowTheProcess/gatekeeper?logo=github&sort=semver)](https://github.com/FollowTheProcess/gatekeeper)
[![CI](https://github.com/FollowTheProcess/gatekeeper/workflows/CI/badge.svg)](https://github.com/FollowTheProcess/gatekeeper/actions?query=workflow%3ACI)
[![codecov](https://codecov.io/gh/FollowTheProcess/gatekeeper/branch/main/graph/badge.svg)](https://codecov.io/gh/FollowTheProcess/gatekeeper)

A custom CI -> AWS auth mechanism powering my personal software catalog 🗄️

---

> [!WARNING]
> **Gatekeeper is personal software, unless your use case exactly matches mine you probably don't want to use it**

## What is it?

Before I can explain what `gatekeeper` is, I need to explain another project 🤓

I host releases of my software (the installable applications anyway) at `get.followtheprocess.codes/{project}/{version}/{artifact}`.

So if you want [gowc] (as an example) on an arm64 macOS you can do:

```bash
curl -L https://get.followtheprocess.codes/gowc/v0.6.4/gowc-v0.6.4-darwin-arm64.tar.gz | tar xz
```

And then:

```bash
./gowc --version
```

Will work 🎉

This works great for things like [homebrew], an excerpt from the gowc [cask] is shown below

```ruby
cask "gowc" do
  version "0.6.4"

  on_macos do
    on_intel do
      sha256 "c50b16c75f0fb939ccd69c3899a22a244f849544997e115eade7c19e1a924785"
      url "https://get.followtheprocess.codes/gowc/v#{version}/gowc-v#{version}-darwin-x86_64.tar.gz",
        verified: "github.com/FollowTheProcess/gowc"
    end
    on_arm do
      sha256 "fd602757fc7db09b4ee3306bde20472c53562427ad4334b25b9ce5200ea5b428"
      url "https://get.followtheprocess.codes/gowc/v#{version}/gowc-v#{version}-darwin-arm64.tar.gz",
        verified: "github.com/FollowTheProcess/gowc"
    end
  end

# Rest of the cask definition...
```

Why do I do this? Mostly because I think it's cool, but also because it reduces lock-in and decouples my releases and installs from e.g. Github releases.

Okay, now on to `gatekeeper`...

This is all powered by AWS S3 and a CloudFront distribution serving it. When I release something, I need to compile the thing, zip it up, then upload it to S3 at the right path.

> [!NOTE]
> [GoReleaser] takes care of almost all of this btw, all I do is own the S3 bucket and CloudFront distribution

And of course, I like to do this with CI/CD. On Github, I can use Github Actions and authenticate to AWS with [OIDC],
assume an IAM Role with `s3:PutObject` for a limited time and we're done 👌🏻

But OIDC yet again ties me into an OIDC-capable place to store my code (Github, Gitlab etc.) If I wanted to move my code to e.g. [tangled] or [codeberg], I couldn't
use this same process to release, not fun!

Enter `gatekeeper`, it _replaces_ OIDC for this use case. Allowing any CI system to authenticate itself using public/private key pairs, then assume a tightly scoped, time bounded
IAM Role allowing it to upload releases.

It's actually even better in some ways! A normal IAM role assumable via OIDC doesn't know where it's going to be assumed from so you have to add a list of approved repos, branches etc. and
it has no knowledge of _which_ file it will be asked to upload so must grant `s3:PutObject` on the whole bucket.

In `gatekeeper`, I know (via the JWT claims) which project and version I'm uploading so can grant `s3:PutObject` _only_ to `{bucket}/{project}/{version}*` so it's _not possible_ for it
to affect anything else in the bucket. 🔒

## How it works

`gatekeeper` is actually 2 separate systems:

- A CLI run in the CI system (or locally)
- A Lambda Function behind a function URL acting as the "auth" server

The flow goes something like this:

1. Generate a public/private key pair with `gatekeeper keys`
2. The private key is stored as a repository secret in the git provider (most of them support this it seems) and exposed as `GATEKEEPER_PRIVATE_KEY`
3. The public key is stored in an SSM parameter scoped to the project (`/releases/{project}/public-key`)
4. When it comes time to release, the CLI is used to request temporary release credentials `gatekeeper auth {project} {version}`
5. The CLI mints and signs (using the private key) a JWT and sends it off to the Lambda. The claims encode project, version etc.
6. The Lambda checks the JWT with the public key and validates the claims just as a traditional auth backend would
7. If the request is authenticated, the Lambda assumes a tightly scoped, time-bounded IAM Role with `s3:PutObject` narrowed to the project and version
8. The Lambda responds with the `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and `AWS_SESSION_TOKEN` values for this session
9. The CLI prints the `export ...` statements so `eval "$(gatekeeper auth project version)"` sources the env vars
10. Run whatever command you need with access to this role (such as `goreleaser release`)
11. The session expires when the JWT does up to a hardcoded maximum of 30 minutes

[homebrew]: https://brew.sh
[OIDC]: https://docs.github.com/en/actions/how-tos/secure-your-work/security-harden-deployments/oidc-in-aws
[gowc]: https://github.com/FollowTheProcess/gowc
[cask]: https://github.com/FollowTheProcess/homebrew-tap/blob/main/Casks/gowc.rb
[GoReleaser]: https://github.com/goreleaser/goreleaser
[tangled]: https://tangled.org
[codeberg]: https://codeberg.org
