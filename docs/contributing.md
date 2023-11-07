# Contributing to Hops

We encourage everyone to help improve hops in whatever way they can. All contributions, large or small are welcome.

If you don't feel able to make changes to the code, there's several other ways to contribute:
- Raising genuine issues with reproduceable bugs is hugely valuable.
- Adding or improving documentation
- Adding test cases (okay this one is code, but you get the drift)

Your change doesn't have to be big to be included. Improving an existing test, adding small test cases, improving readability are all helpful ways to get started.

## Code of conduct

We expect all discussion to be respectful, inclusive and polite and we will enforce this to protect the community. Being kind is just as important as being right. Equally we expect all issues raised and features requested to be given due respect by maintainers.

To avoid disappointment, it is always better to start a discussion before contributing a PR. This way we can agree an implementation and also make you aware of any internal roadmap items that may impact the change.

Smaller PRs are preferred as they make reviewing contributed code much simpler. A feature or change that is naturally large may need to be broken into smaller pieces. This should be identified in discussion prior to starting.

Whilst we have some areas of the codebase that need improved test coverage, new contributions will require full tests to be included (we don't want to increase our backlog of test cases to write)

If you haven't contributed before, it's likely easiest to make a small change first (even if you have a really big change you'd like to make). This gives us a chance to get to know each other, and gives you a chance to understand how we work.

The most important rule of all: Be awesome to each other.


## Dev setup

To work on hops, you'll need to build the console first. Follow the instructions in `/console` for a production build.

If working on the console specifically, you can start hops independently and run the console in dev mode.

## Testing

To run tests, from the root of the repo run:
`go test ./... -test.v`

### Test style guide

#### Test that code produces the correct result, not that it works a certain way

Tests should never be based on 'was x thing called'.

This style of testing is very brittle. Tests can be invalidated by making changes to the code that have exactly equivalent behaviour.

The only exception is if the output of a unit under test is expressed _only_ via some side-effect - such as an HTTP call being made. Even then, there's often better ways to construct the test (e.g. mock servers that error on unexpected input)

#### Use assert/require

We use the [testify](https://pkg.go.dev/github.com/stretchr/testify) package for assertions.

Whilst some golang style guides recommend against assertions, we find they make simple and readable tests that are understood by a wide range of programmers.

`assert` should be used in most cases as it allows a test to continue and reveal more errors in a single pass

`require` is used when a condition not being met means the rest of the test will _always_
fail or panic (e.g. a boostrap function fails, the result of which is required for the remaining assertions)

If in doubt, use `assert`

### Use table based testing

Table based testing makes it much easier to add new, similar test cases. We do have a several tests that need to be re-written to use this approach, but please don't add more.
Not all tests _should_ be table based, but where it makes sense it is required for future contributions.
