# Hops configuration language (.hops)


## Dev setup (for developing hops itself)

To work on hops, you'll need to build the console first. Follow the instructions in `/console` for a production build.

If working on the console specifically, you can start hops independently and run the console in dev mode. Note: There's an outstanding task to make this workflow functional again. Broken due to change where we assume backend is relative to UI's current domain/port.


## Testing

To run tests, from the root of the repo run:
`go test ./... -test.v`

Currently, running tests requires your `local` secret store to be populated with
your Hiphops.io refresh token (see [Connecting services](#connecting-services)).

Adding a persistent, zero-permission token as a fixture is on the todo list which will remove this step.


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

## Builds

### Linux

Linux arm64 executable can be tested on an arm mac using the following command from the directory with the executable in it.

```bash
docker build . -f docker/Dockerfile
docker run -v ~/.hops:/root/.hops --rm sha256:<big number appearing at the end of the build> start --address=0.0.0.0:8916
```

App will be available at `localhost:8916`.

*Note*: if you can't reach the app, you will probably need to port forward to another port: `docker run -p 8200:8916 ...`

### Macos

#### Running

In order to run `hops` downloaded from github, you have to override gatekeeper.

To do this, go to the directory where you downloaded `hops` in `Finder`, and Option + Right Click on the file, and select `Open`. This will pop up a dialog that will allow you to open the file.

Once this has been done, `hops` can be run normally from the command line.

#### Signing

Info about signing

```
codesign -dv -r- ~/Downloads/hops
```

#### Apple development certificates

We store the Developer ID Application certificate in GitHub Actions as an encrypted secret. If that is missing, do the following:

First we export the Developer ID Application certificate and private key into a `.p12` file. Here's how to do it:

1. Open the Keychain Access app on your Mac.
2. Under the "My Certificates" category, find your Developer ID Application certificate.
3. When you find it, there should be an arrow to the left of the name. Click on this arrow and it should show the private key associated with the certificate underneath it.
4. Select both the certificate and the key. You can do this by clicking on the certificate, then holding the Command key and clicking on the key.
5. Right-click (or Control-click) on the highlighted certificate and key and select "Export 2 items...".
6. Choose the .p12 format and save the file.
7. You'll be asked to create a password. This password is used to protect the exported items.

Now, you can create a GitHub secret to store this file:

1. Convert the `.p12` file into a base64 string (this is needed because GitHub secrets can only store text):

   ```bash
   base64 -i YourCertificate.p12 -o YourCertificate_p12_base64.txt
   ```

2. Open `YourCertificate_p12_base64.txt` and copy the entire base64 string.

3. Go to your GitHub repository, and click on "Settings" -> "Secrets" -> "New repository secret".

4. Give the secret a name: `APPLE_CODE_SIGNING_CERTIFICATE_P12_BASE64`, and paste the base64 string into the "Value" field.

5. You should also create another secret for the password you used when exporting the `.p12` file. Call this `APPLE_CODE_SIGNING_CERTIFICATE_P12_PASSWORD`, and paste the password in here.

Additionally you will need to set the following secrets in the Githop repo:

- `APPLE_LOGIN`: login email for the apple developer account
- `APPLE_TEAM_ID`: can be found at at this location: https://developer.apple.com/account/resources/certificates at the top right after organisation name and below the user name
- `APPLE_CODE_SIGNING_DEVELOPER_ID`: "Developer ID Application: Hiphops.io ltd ($APPLE_TEAM_ID)" where the `$APPLE_TEAM_ID` is the value just above. (See)[https://wiki.lazarus.freepascal.org/Code_Signing_for_macOS]
- `APPLE_NOTARYTOOL_PASSWORD`: This is an app specific password for the notarytool. It can be created here: https://appleid.apple.com/account/manage under Application Specific Passwords.
